.PHONY: dev build web test fmt vet clean

BIN := bin/crew

dev:
	go run ./cmd/crew

web:
	cd web && npm ci && npm run build
	rm -rf internal/webui/dist
	mkdir -p internal/webui/dist
	cp -R web/dist/. internal/webui/dist/

build: web
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/crew

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

clean:
	rm -rf bin internal/webui/dist web/dist
	mkdir -p internal/webui/dist
	touch internal/webui/dist/.gitkeep
