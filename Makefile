APP_NAME := agentdeck
BIN_DIR := bin
WEB_DIR := web
EMBED_DIST := internal/webembed/dist
# sudo 会清掉 PATH，用当前用户能找到的 go 绝对路径
GO := $(shell command -v go)

.DEFAULT_GOAL := help

.PHONY: help all build web backend clean run dev-api dev-api-sudo dev-web embed-stub

help:
	@echo "AgentDeck Make 目标"
	@echo ""
	@echo "  make help          显示本帮助"
	@echo "  make dev-api       启动后端（:8080）"
	@echo "  make dev-api-sudo  以 root 启动后端（进程详情/防火墙需要）"
	@echo "  make dev-web       启动前端（:5173，/api 代理到后端）"
	@echo "  make web           构建前端并嵌入到 internal/webembed/dist"
	@echo "  make backend       仅编译 Go 可执行文件到 bin/"
	@echo "  make build         构建前端并编译后端"
	@echo "  make run           构建后启动 bin/$(APP_NAME)"
	@echo "  make clean         清理构建产物与 data/"
	@echo ""

all: build

# go:embed 要求 dist 在编译时存在；正式页面由 make web 覆盖，此处只保证目录可编译
embed-stub:
	@mkdir -p $(EMBED_DIST)
	@test -f $(EMBED_DIST)/index.html || printf '%s\n' '<!DOCTYPE html><html><body>请先执行 make web</body></html>' > $(EMBED_DIST)/index.html

web:
	cd $(WEB_DIR) && pnpm install && pnpm build
	rm -rf $(EMBED_DIST)
	mkdir -p $(EMBED_DIST)
	cp -a $(WEB_DIR)/dist/. $(EMBED_DIST)/

backend: embed-stub
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

build: web backend

run: build
	./$(BIN_DIR)/$(APP_NAME)

# 分开开发：两个终端分别执行
dev-api: embed-stub
	$(GO) run ./cmd/server

# 不要 sudo make；在目标里 sudo，并补上 /usr/sbin（ufw/iptables 需要）
dev-api-sudo: embed-stub
	sudo env PATH="/usr/sbin:/sbin:$(PATH)" HOME="$(HOME)" GOCACHE="$(shell $(GO) env GOCACHE)" GOMODCACHE="$(shell $(GO) env GOMODCACHE)" $(GO) run ./cmd/server

dev-web:
	cd $(WEB_DIR) && pnpm install && pnpm dev

clean:
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(EMBED_DIST) data
	mkdir -p $(EMBED_DIST)
	printf '%s\n' '<!DOCTYPE html><html><body>build required</body></html>' > $(EMBED_DIST)/index.html
