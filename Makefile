APP_NAME := agentdeck
BIN_DIR := bin
WEB_DIR := web
EMBED_DIST := internal/webembed/dist

.PHONY: all build web backend clean run dev-web

all: build

web:
	cd $(WEB_DIR) && pnpm install && pnpm build
	rm -rf $(EMBED_DIST)
	mkdir -p $(EMBED_DIST)
	cp -a $(WEB_DIR)/dist/. $(EMBED_DIST)/

backend:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

build: web backend

run: build
	./$(BIN_DIR)/$(APP_NAME)

dev-web:
	cd $(WEB_DIR) && pnpm dev

clean:
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(EMBED_DIST) data
	mkdir -p $(EMBED_DIST)
	printf '%s\n' '<!DOCTYPE html><html><body>build required</body></html>' > $(EMBED_DIST)/index.html
