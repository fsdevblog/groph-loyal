// Package accrual обрабатывает начисление баллов за заказы через внешний API.
package accrual

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/fsdevblog/groph-loyal/internal/transport/accrual/client"
	"github.com/shopspring/decimal"

	"github.com/sirupsen/logrus"

	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

const (
	defaultServiceTimeout         = 3 * time.Second
	defaultAPITimeout             = 10 * time.Second
	defaultLimitPerIteration uint = 100
	defaultAccrualWorkers    uint = 10
)

// Processor обрабатывает начисления баллов за заказы через внешний API начислений.
type Processor struct {
	client            Client
	svs               Servicer
	l                 *logrus.Entry
	limitPerIteration uint
	accrualWorkers    uint
}

// New создает новый экземпляр процессора обработки начислений.
func New(svs Servicer, apiBaseURL string, l *logrus.Logger) *Processor {
	loggerEntry := l.WithFields(logrus.Fields{
		"component": "accrual",
		"module":    "processor",
	})

	return &Processor{
		svs:               svs,
		client:            client.New(apiBaseURL),
		l:                 loggerEntry,
		limitPerIteration: defaultLimitPerIteration,
		accrualWorkers:    defaultAccrualWorkers,
	}
}

// SetLimitPerIteration устанавливает кол-во заказов, обрабатываемых в одной итерации обработчика.
func (p *Processor) SetLimitPerIteration(limit uint) *Processor {
	p.limitPerIteration = limit
	return p
}

// SetAccrualWorkers устанавливает кол-во воркеров работающих с заказами.
func (p *Processor) SetAccrualWorkers(workers uint) *Processor {
	p.accrualWorkers = workers
	return p
}

// Run запускает обработку заказов в бесконечном цикле до отмены контекста.
//
// Алгоритм работы:
//  1. В каждой итерации цикла, запрашивает через сервисный слой список заказов для обработки. Объем списка лимитируется
//     через SetLimitPerIteration.
//  2. Для каждой итерации создаются N воркеров (кол-во настраивается через SetAccrualWorkers)
//     которые, в свою очередь, делают запросы на API сервиса начисления баллов.
//  3. Результат работы отправляется через сервисный слой.
func (p *Processor) Run(ctx context.Context) {
	p.l.WithFields(logrus.Fields{
		"limitPerIteration": p.limitPerIteration,
		"accrualWorkers":    p.accrualWorkers,
	}).Info("Starting")

	for {
		select {
		case <-ctx.Done():
			p.l.Info("Got stop signal, exiting...")
			return
		default:
			if err := p.process(ctx); err != nil {
				if !errors.Is(err, ErrNoOrders) {
					p.l.WithError(err).Error("process error")
				}
				time.Sleep(time.Second) // небольшая пауза чтоб не заддосить БД.
			}
		}
	}
}

// process выполняет цикл обработки заказов: получение списка, запрос данных через API и обновление информации.
// Возвращает ошибку в случае проблем или ErrNoOrders если нет заказов для обработки.
func (p *Processor) process(ctx context.Context) error {
	orders, ordersErr := p.produce(ctx)

	if ordersErr != nil {
		return fmt.Errorf("process: %w", ordersErr)
	}

	results := p.runWorkers(ctx, orders)
	if len(results) == 0 {
		return nil
	}

	var updateArgs = make([]service.UpdateAccrualArgs, 0, len(results))
	for _, result := range results {
		// согласно ТЗ статус REGISTERED не поддерживается системой лояльности, поэтому его мы пропускаем,
		// и воркер его обработает при следующей попытке.
		if result.Status == client.StatusRegistered {
			continue
		}
		updateArgs = append(updateArgs, service.UpdateAccrualArgs{
			Error:   result.Error,
			OrderID: result.Order.ID,
			Attempt: result.Attempt,
			Status:  domain.OrderStatusType(result.Status),
			Accrual: result.Accrual,
		})
	}

	reqCtx, cancel := context.WithTimeout(ctx, defaultServiceTimeout)
	defer cancel()

	if updErr := p.svs.UpdateAccrual(reqCtx, updateArgs); updErr != nil {
		return fmt.Errorf("process: %s", updErr.Error())
	}

	return nil
}

// workerResult представляет результат работы воркера по запросу начислений.
type workerResult struct {
	WorkerID uint
	Attempt  uint
	Order    *domain.Order
	Error    error
	Status   client.StatusType
	Accrual  decimal.Decimal
}

// runWorkers запускает параллельных воркеров для получения данных о начислениях и ожидает конца их работы.
// Реализует паттерн fan-out/fan-in для параллельной обработки запросов.
func (p *Processor) runWorkers(ctx context.Context, orders []domain.Order) []workerResult {
	var taskCh = make(chan *domain.Order, len(orders))

	for _, order := range orders {
		taskCh <- &order
	}
	close(taskCh)

	wg := new(sync.WaitGroup)
	wg.Add(int(p.accrualWorkers)) // nolint:gosec

	var resultCh = make(chan *workerResult, len(orders))

	for i := range p.accrualWorkers {
		go p.worker(ctx, wg, i+1, taskCh, resultCh)
	}
	wg.Wait()

	close(resultCh)

	var results = make([]workerResult, 0, len(orders))
	for result := range resultCh {
		l := p.l.WithFields(logrus.Fields{
			"worker":  result.WorkerID,
			"orderID": result.Order.ID,
			"attempt": result.Attempt + 1,
		})
		if result.Error != nil {
			l.WithError(result.Error).Error("get accrual for order")
			results = append(results, workerResult{
				Order:   result.Order,
				Attempt: result.Attempt,
				Error:   result.Error,
			})
		} else {
			l.WithField("accrual", result.Accrual).Info("Success")
			results = append(results, workerResult{
				Order:   result.Order,
				Status:  result.Status,
				Accrual: result.Accrual,
				Attempt: result.Attempt,
				Error:   nil,
			})
		}
	}
	return results
}

// worker обрабатывает заказы из канала, запрашивает данные через API и отправляет результаты.
func (p *Processor) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	workerID uint,
	taskCh <-chan *domain.Order,
	resultCh chan<- *workerResult,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-taskCh:
			if !ok {
				return
			}
			resultCh <- p.processWorkerTask(ctx, workerID, task)
		}
	}
}

// processWorkerTask делает запрос на API системы начисления, в случае получения ошибки 429, ждет N секунд указанные
// в заголовке ответа.
func (p *Processor) processWorkerTask(ctx context.Context, workerID uint, task *domain.Order) *workerResult {
	for {
		reqCtx, cancel := context.WithTimeout(ctx, defaultAPITimeout)
		resp, err := p.client.GetOrderAccrual(reqCtx, task.OrderCode)
		cancel()

		// Проверяем ошибку на TooManyRequestError для повторной попытки
		if err != nil {
			result := workerResult{
				WorkerID: workerID,
				Order:    task,
				Attempt:  task.Attempts,
			}
			var tooManyReq *client.TooManyRequestError
			if errors.As(err, &tooManyReq) {
				// Проверяем отмену контекста перед спячкой
				select {
				case <-ctx.Done():
					result.Error = ctx.Err()
					return &result
				case <-time.After(tooManyReq.RetryAfter):
					// После паузы делаем повторную попытку
					continue
				}
			} else {
				result.Error = err
				return &result
			}
		}

		return &workerResult{
			WorkerID: workerID,
			Order:    task,
			Status:   resp.Status,
			Attempt:  task.Attempts,
			Accrual:  resp.Accrual,
		}
	}
}

// produce получает список заказов для обработки начислений.
// Возвращает ErrNoOrders, если заказы отсутствуют.
func (p *Processor) produce(ctx context.Context) ([]domain.Order, error) {
	produceCtx, cancel := context.WithTimeout(ctx, defaultServiceTimeout)
	defer cancel()

	orders, ordersErr := p.svs.OrdersForAccrualMonitoring(produceCtx, p.limitPerIteration)
	if ordersErr != nil {
		return nil, fmt.Errorf("produce: %w", ordersErr)
	}

	if len(orders) == 0 {
		return nil, ErrNoOrders
	}
	return orders, nil
}
