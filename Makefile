deps:
	go mod download
	go mod tidy
.PHONY: deps

build: deps
	go build -o ./bin/jetapi ./cmd/jetapi
.PHONY: build

run: build
	./bin/jetapi
.PHONY: run

run-dev:
	air --build.cmd "go build -o ./bin/jetapi ./cmd/jetapi" --build.bin "./bin/jetapi"
.PHONY: run-dev

clean:
	rm -r bin/
	rm -r tmp/
	rm -r cmd/jetapi/tmp
.PHONY: clean

docker-build:
	docker-compose build
.PHONY: docker-build

docker-build-no-cache:
	docker-compose build --no-cache
.PHONY: docker-build-no-cache

docker-clean:
	docker-compose down
	docker builder prune -af
.PHONY: docker-clean
