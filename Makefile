build:
	go build -o ./bin/jetapi ./cmd/jetapi
.PHONY: build

run: build
	./bin/jetapi
.PHONY: run

run-dev:
	air --build.cmd "go build -o ./bin/jetapi ./cmd/jetapi" --build.bin "./bin/jetapi"
.PHONY: run-dev

run-prod: build
	PORT=8154 nohup ./bin/jetapi > ./bin/jetapi.log 2>&1 & echo $$! > ./bin/jetapi.pid
	@echo "Servidor iniciado em background na porta 8154 (PID: $$(cat ./bin/jetapi.pid))"
.PHONY: run-prod

stop-prod:
	@if [ -f ./bin/jetapi.pid ]; then \
		kill $$(cat ./bin/jetapi.pid) 2>/dev/null && rm ./bin/jetapi.pid && echo "Servidor parado"; \
	else \
		echo "PID não encontrado. O servidor pode não estar rodando."; \
	fi
.PHONY: stop-prod

clean:
	rm -r bin/
	rm -r tmp/
	rm -r cmd/jetapi/tmp
.PHONY: clean
