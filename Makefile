FRONTEND_DIR := frontend
BINARY := bin/cups-web
GO_TAGS := frontend_dist

.PHONY: all frontend build backend clean docker-build test
all: build

frontend:
	@echo "Building frontend..."
	@if command -v bun >/dev/null 2>&1; then \
		cd $(FRONTEND_DIR) && rm -rf node_modules && \
		if bun install --frozen-lockfile && bun run build; then \
			true; \
		else \
			echo "bun build failed, falling back to npm ci..." >&2; \
			rm -rf node_modules && npm cache clean --force >/dev/null 2>&1 && npm ci && npm run build; \
		fi; \
	else \
		cd $(FRONTEND_DIR) && rm -rf node_modules && npm cache clean --force >/dev/null 2>&1 && npm ci && npm run build; \
	fi

build: frontend backend

backend:
	@echo "Building Go binary..."
	go build -tags $(GO_TAGS) -o $(BINARY) ./cmd/server

test:
	@echo "Running frontend build and Go tests..."
	$(MAKE) frontend
	go test ./...

clean:
	rm -f $(BINARY)

docker-build:
	docker build -t cups:latest -f cups/Dockerfile cups
	docker build -t cups-web:latest -f Dockerfile .
