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
	@mkdir -p ./bin
	@if [ -f ./bin/jetapi.pid ]; then \
		if ps -p $$(cat ./bin/jetapi.pid) > /dev/null 2>&1; then \
			echo "Servidor já está rodando (PID: $$(cat ./bin/jetapi.pid))"; \
			exit 1; \
		else \
			rm ./bin/jetapi.pid; \
		fi; \
	fi
	@PROJECT_DIR=$$(pwd) && cd $$PROJECT_DIR && PORT=8154 nohup ./bin/jetapi > ./bin/jetapi.log 2>&1 & echo $$! > ./bin/jetapi.pid
	@sleep 2
	@if [ -f ./bin/jetapi.pid ] && ps -p $$(cat ./bin/jetapi.pid) > /dev/null 2>&1; then \
		echo "Servidor iniciado em background na porta 8154 (PID: $$(cat ./bin/jetapi.pid))"; \
		echo "Logs disponíveis em: ./bin/jetapi.log"; \
		echo "Use 'make logs-prod' para ver os logs em tempo real"; \
		echo "Use 'make status-prod' para verificar o status"; \
	else \
		echo "Erro ao iniciar servidor. Verifique os logs:"; \
		if [ -f ./bin/jetapi.log ]; then \
			tail -30 ./bin/jetapi.log; \
		else \
			echo "Arquivo de log não encontrado"; \
		fi; \
		[ -f ./bin/jetapi.pid ] && rm ./bin/jetapi.pid; \
		exit 1; \
	fi
.PHONY: run-prod

stop-prod:
	@if [ -f ./bin/jetapi.pid ]; then \
		kill $$(cat ./bin/jetapi.pid) 2>/dev/null && rm ./bin/jetapi.pid && echo "Servidor parado"; \
	else \
		echo "PID não encontrado. O servidor pode não estar rodando."; \
	fi
.PHONY: stop-prod

logs-prod:
	@if [ -f ./bin/jetapi.log ]; then \
		tail -f ./bin/jetapi.log; \
	else \
		echo "Arquivo de log não encontrado: ./bin/jetapi.log"; \
	fi
.PHONY: logs-prod

status-prod:
	@if [ -f ./bin/jetapi.pid ]; then \
		if ps -p $$(cat ./bin/jetapi.pid) > /dev/null 2>&1; then \
			echo "Servidor está rodando (PID: $$(cat ./bin/jetapi.pid))"; \
			echo "Últimas 10 linhas do log:"; \
			tail -10 ./bin/jetapi.log 2>/dev/null || echo "Sem logs disponíveis"; \
		else \
			echo "PID encontrado mas processo não está rodando. Limpando PID."; \
			rm ./bin/jetapi.pid; \
		fi; \
	else \
		echo "Servidor não está rodando (nenhum PID encontrado)"; \
	fi
.PHONY: status-prod

clean:
	rm -r bin/
	rm -r tmp/
	rm -r cmd/jetapi/tmp
.PHONY: clean
