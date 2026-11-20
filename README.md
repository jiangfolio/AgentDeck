# AgentDeck

跨平台单可执行文件脚手架：Gin + SQLite + Vue3 + antdv-next + Tailwind + 国密（SM2/SM3/SM4）。

## 特性

- 无外部配置文件，直接运行可执行文件
- SQLite 自动建库建表，首次启动种子管理员 `admin / admin123`
- 服务端 Session（Cookie），不使用 JWT
- **仅登录**走国密加密传输（SM2 协商 + SM4 包体）；日常 API 明文 JSON
- JSON 字段统一 **snake_case**
- 业务统一放在 `internal/modular`（领域 → 模块）；基础设施与横切能力常规放置
- 前端仅 `views` 按领域分，其余按常规分层
- 前端：Vue3 + TS + Vite + antdv-next + Tailwind + IconPark（离线）

## 目录要点

```text
internal/
  modular/                    # 业务领域 → 模块
    auth/login|user|session/
    crypto/key/
    system/host/
    agent/process|cron|startup/
  pkg/platform/               # 跨平台适配（进程/crontab|schtasks/自启）
  middleware/ pkg/ infra/ server/ config/ webembed/
web/src/
  views/auth/ views/home/ views/agent/
  api/ stores/ utils/ layout/ router/ composables/
```

## 开发

```bash
# 后端
go run ./cmd/server

# 前端（另开终端，使用 pnpm）
cd web && pnpm install && pnpm dev
```

前端开发服务器默认 `http://127.0.0.1:5173`，`/api` 代理到 `:8080`。

## 构建与运行

```bash
make build
./bin/agentdeck
```

访问 `http://127.0.0.1:8080`。数据库文件位于可执行文件同目录 `data/agentdeck.db`。

## 交叉编译

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/agentdeck.exe ./cmd/server
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/agentdeck-darwin ./cmd/server
```

先执行 `make web` 嵌入前端产物后再交叉编译后端。
