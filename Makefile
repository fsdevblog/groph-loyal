DATABASE_DSN=postgres://study1-user:123123123@localhost:5435/postgres?sslmode=disable

CMD_DIR=cmd/gophermart

# Название бинарного файла
BINARY=gophermart

# Создать миграцию
migrate-create:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "Укажите имя миграции в формате: make migrate-create <name>"; \
		exit 1; \
	fi
	migrate create -ext sql -dir internal/db/migrations -seq $(filter-out $@,$(MAKECMDGOALS))

%:
	@:


# Миграция вверх
migrate-up:
	migrate -database "postgres://study1-user:123123123@localhost:5436/postgres?sslmode=disable" -path ./internal/db/migrations up

# Миграция вниз
migrate-down:
	migrate -database "postgres://study1-user:123123123@localhost:5436/postgres?sslmode=disable" -path ./internal/db/migrations down 1

test:
	go test ./... -v

vet:
	go vet -vettool=./statictest ./...

db-up:
	docker compose up -d postgres
db-down:
	docker compose down --remove-orphans

# Билд приложения
build:
	cd $(CMD_DIR) && go build -o $(BINARY) *.go

run:
	cd $(CMD_DIR) && ./$(BINARY) -d "postgres://study1-user:123123123@localhost:5436/postgres?sslmode=disable" -m "../../internal/db/migrations"

run-accrual:
	cmd/accrual/accrual_linux_amd64 -d postgres://study1-user:123123123@localhost:5436/accrual?sslmode=disable -a :8081

build-run: build run

sqlc-generate:
	rm -rf internal/db/repository/sqlc/sqlcgen/* && docker compose run --rm sqlc

auto-tests:
	./gophermarttest -test.v -test.run=.* -gophermart-binary-path=./cmd/gophermart/gophermart -accrual-binary-path=./cmd/accrual/accrual_linux_amd64 -accrual-database-uri="postgres://study1-user:123123123@localhost:5436/accrual?sslmode=disable" -gophermart-database-uri="postgres://study1-user:123123123@localhost:5436/postgres?sslmode=disable" -gophermart-host=localhost -accrual-host=localhost -gophermart-port=8080 -accrual-port=8081