.PHONY: run build generate swagger fmt vet test up down restart

run:
	go run ./cmd/api

# Kills whatever's listening on the API's HTTP port (a stale `go run` or
# `bin/api` left over from before a code change), rebuilds the binary, and
# starts it in the background.
restart:
	@PORT=$${HTTP_PORT:-8090}; \
	PID=$$(lsof -ti tcp:$$PORT); \
	if [ -n "$$PID" ]; then echo "stopping process on port $$PORT (pid $$PID)"; kill $$PID; sleep 1; fi
	@CGO_ENABLED=0 go build -o bin/api ./cmd/api
	@nohup ./bin/api > api.log 2>&1 & \
	echo "started api (pid $$!) — logs: tail -f api.log"

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

generate:
	cd ent && go generate ./...

swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

up:
	docker compose up -d postgres redis

down:
	docker compose down
