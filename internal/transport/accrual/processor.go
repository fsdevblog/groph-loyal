// Package accrual содержит функциональность для обработки начисления баллов за заказы.
// Модуль предоставляет процессор, который в фоновом режиме обрабатывает заказы, запрашивает информацию
// о начислениях через внешний API и обновляет состояние заказов с полученными данными.
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
	// defaultServiceTimeout задает стандартный таймаут для операций сервиса (3 секунды).
	defaultServiceTimeout = 3 * time.Second
	// defaultAPITimeout задает стандартный таймаут для запросов к внешнему API (10 секунд).
	defaultAPITimeout = 10 * time.Second

	// Значения по умолчанию для настроек процессора.

	// defaultLimitPerIteration определяет максимальное количество заказов, обрабатываемых за одну итерацию.
	defaultLimitPerIteration uint = 100
	// defaultAccrualWorkers устанавливает количество параллельных обработчиков для запросов начислений.
	defaultAccrualWorkers uint = 10
)

// Processor управляет процессом получения и обновления информации о начислениях баллов за заказы.
// Работает в фоновом режиме, обрабатывая заказы пакетами и взаимодействуя с внешним API начислений.
type Processor struct {
	client            Client        // Клиент для взаимодействия с API начислений
	svs               Servicer      // Сервис для работы с заказами
	l                 *logrus.Entry // Логгер с контекстными полями компонента
	limitPerIteration uint          // Максимальное количество заказов за одну итерацию
	accrualWorkers    uint          // Количество параллельных обработчиков
}

// ProcessorOptions содержит опции для настройки процессора обработки начислений.
type ProcessorOptions struct {
	LimitPerIteration uint // Максимальное количество заказов за одну итерацию
	AccrualWorkers    uint // Количество параллельных обработчиков
}

// NewProcessor создает новый экземпляр процессора обработки начислений с заданными параметрами.
// Позволяет настроить поведение процессора через функциональные опции.
//
// Параметры:
//   - svs: сервис для работы с заказами
//   - l: экземпляр логгера
//   - opts: функциональные опции для настройки процессора
//
// Возвращает настроенный экземпляр процессора.
func NewProcessor(svs Servicer, apiBaseURL string, l *logrus.Logger, opts ...func(*ProcessorOptions)) *Processor {
	options := ProcessorOptions{
		LimitPerIteration: defaultLimitPerIteration,
		AccrualWorkers:    defaultAccrualWorkers,
	}

	loggerEntry := l.WithFields(logrus.Fields{
		"component": "accrual",
		"module":    "processor",
	})

	for _, opt := range opts {
		opt(&options)
	}

	return &Processor{
		svs:               svs,
		client:            client.NewHTTPClient(apiBaseURL),
		l:                 loggerEntry,
		limitPerIteration: options.LimitPerIteration,
		accrualWorkers:    options.AccrualWorkers,
	}
}

// Run запускает бесконечный цикл обработки заказов, ожидающих начисления баллов.
// Цикл продолжается до отмены контекста. При возникновении ошибок в процессе обработки,
// процессор логирует их и продолжает работу. Между каждой итерацией происходит небольшая пауза.
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом процессора
func (p *Processor) Run(ctx context.Context) {
	p.l.Info("Starting")

	for {
		select {
		case <-ctx.Done():
			p.l.Info("Got stop signal, exiting...")
			return
		default:
			if err := p.process(ctx); err != nil {
				p.l.WithError(err).Error("process error")
			}
			time.Sleep(time.Second)
		}
	}
}

// process выполняет полный цикл обработки заказов: получение списка заказов для обработки,
// запуск воркеров для получения данных о начислениях и обновление информации о заказах.
//
// Алгоритм работы:
// 1. Получает заказы, требующие обработки начислений
// 2. Запускает параллельные воркеры для запроса данных о начислениях
// 3. Обновляет информацию о заказах с полученными данными
//
// Возвращает ошибку, если возникла проблема в процессе обработки (кроме случая отсутствия заказов).
func (p *Processor) process(ctx context.Context) error {
	orders, ordersErr := p.produce(ctx)

	if ordersErr != nil {
		if errors.Is(ordersErr, ErrNoOrders) {
			return nil
		}
		return fmt.Errorf("process: %s", ordersErr.Error())
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
			OrderID: result.OrderID,
			Status:  domain.OrderStatusType(result.Status),
			Accrual: result.Accrual,
		})
	}

	if updErr := p.svs.UpdateAccrual(ctx, updateArgs); updErr != nil {
		return fmt.Errorf("process: %s", updErr.Error())
	}

	return nil
}

// workerResult представляет результат работы воркера по запросу начислений.
// Содержит информацию о заказе, возможной ошибке и данные о начислении.
type workerResult struct {
	OrderID int64             // Идентификатор заказа. Пуст если есть ошибка
	Error   error             // Ошибка, возникшая при запросе данных начисления
	Status  client.StatusType // Статус заказа. Пустая строка если ошибка
	Accrual decimal.Decimal   // Сумма к начислению. Zero если ошибка
}

// runWorkers запускает несколько параллельных воркеров для получения данных о начислениях
// для списка заказов. Реализует паттерн fan-out/fan-in для параллельной обработки запросов.
//
// Алгоритм работы:
// 1. Создает канал задач и заполняет его заказами для обработки
// 2. Запускает несколько горутин-воркеров для параллельного выполнения запросов
// 3. Собирает результаты обработки из канала результатов
// 4. Преобразует успешные результаты в структуры для обновления заказов
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом воркеров
//   - orders: список заказов для обработки.
//
// Возвращает срез структур с данными для обновления заказов.
func (p *Processor) runWorkers(ctx context.Context, orders []domain.Order) []workerResult {
	var taskCh = make(chan *domain.Order, len(orders))

	for _, order := range orders {
		taskCh <- &order
	}
	close(taskCh)

	wg := new(sync.WaitGroup)
	wg.Add(int(p.accrualWorkers)) // nolint:gosec

	var resultCh = make(chan *workerResult, len(orders))

	for range p.accrualWorkers {
		p.worker(ctx, wg, taskCh, resultCh)
	}
	wg.Wait()

	close(resultCh)

	var results = make([]workerResult, 0, len(orders))
	for result := range resultCh {
		if result.Error != nil {
			p.l.WithError(result.Error).
				Errorf("get accrual for order %d", result.OrderID)
			results = append(results, workerResult{
				OrderID: result.OrderID,
				Error:   result.Error,
			})
		} else {
			results = append(results, workerResult{
				OrderID: result.OrderID,
				Status:  result.Status,
				Accrual: result.Accrual,
				Error:   nil,
			})
		}
	}
	return results
}

// worker запускает воркер, который читает заказы из канала задач, запрашивает данные
// о начислениях через клиент API и отправляет результаты в канал результатов.
// Воркер завершается при закрытии канала задач или отмене контекста.
//
// Параметры:
//   - ctx: контекст для управления жизненным циклом воркера
//   - wg: WaitGroup для синхронизации завершения всех воркеров
//   - taskCh: канал задач с заказами для обработки
//   - resultCh: канал для отправки результатов обработки
func (p *Processor) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
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

			// Создаем контекст с таймаутом для запроса к API
			reqCtx, cancel := context.WithTimeout(ctx, defaultAPITimeout)
			resp, err := p.client.GetOrderAccrual(reqCtx, task.OrderCode)
			cancel()
			if err != nil {
				resultCh <- &workerResult{
					OrderID: task.ID,
					Error:   err,
				}
				return
			}

			resultCh <- &workerResult{
				OrderID: task.ID,
				Status:  resp.Status,
				Accrual: resp.Accrual,
			}
		}
	}
}

// produce получает из сервиса список заказов, ожидающих обработки начислений.
// Запрос к сервису выполняется с ограниченным таймаутом.
//
// Если заказы для обработки отсутствуют, возвращает специальную ошибку ErrNoOrders.
// При других ошибках в работе сервиса возвращает ошибку с контекстом.
//
// Параметры:
//   - ctx: контекст для управления запросом
//
// Возвращает:
//   - список заказов для обработки
//   - ошибку, если запрос не удался или заказы отсутствуют
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
