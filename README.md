<div align="center">

# AgentDeck

**基础设施管理面板**

Gin + SQLite + Vue3 · 单可执行文件 · 主机、进程、任务、磁盘与防火墙

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Gin](https://img.shields.io/badge/Gin-1.12-blue)](https://gin-gonic.com/)
[![Vue](https://img.shields.io/badge/Vue-3.5-brightgreen?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![antdv-next](https://img.shields.io/badge/antdv--next-1.8-1677FF)](https://github.com/antdv-next/antdv-next)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

</div>

> 在自己的机器上开一个面板：看资源、管进程和计划任务、改防火墙、看磁盘。前端打进同一个可执行文件，拷过去就能跑。

---

## 目录

- [特性](#特性)
- [技术栈](#技术栈)
- [目录结构](#目录结构)
- [环境依赖](#环境依赖)
- [快速开始](#快速开始)
- [构建](#构建)
- [说明](#说明)
- [许可证](#许可证)

## 特性

- **主机概况**：系统信息、CPU、内存、交换区、磁盘占用
- **实时监控**：负载、CPU、内存、磁盘 IO、网卡流量，以及当前最吃 CPU 的进程。曲线只留在页面里，不落库
- **进程**：按父子关系展开，可结束进程或进程树；详情含内存、打开的文件、环境变量和网络连接
- **定时任务 / 开机自启**：命令用编辑器编写，可先试运行
- **磁盘**：分区容量与使用率。系统盘只查看
- **防火墙**：查看并修改规则，支持开关、增删改、导入导出。Linux 走 ufw，Windows 走系统防火墙
- **应用目录**：分类、搜索、是否已安装。安装流程尚未接入
- **单文件运行**：无独立配置文件；SQLite 放在可执行文件旁的 `data/agentdeck.db`
- **登录**：Cookie 会话。只有登录请求走国密，其余接口是 JSON

## 技术栈

| 层 | 选型 |
|----|------|
| 服务 | Go · Gin · GORM · SQLite · gopsutil |
| 页面 | Vue 3 · TypeScript · Vite · antdv-next · Tailwind · ECharts |
| 登录传输 | SM2 协商 + SM4 包体 |

## 目录结构

```text
cmd/server                 启动入口
internal/modular           登录、主机、进程、任务、自启、监控、防火墙、磁盘、应用目录
internal/pkg/platform      各系统命令差异（计划任务、自启、防火墙）
internal/webembed          嵌入的前端产物
web/src/views              页面对应上面这些能力
```

## 环境依赖

- Go 1.27+
- Node.js 与 pnpm（只在构建或开发前端时需要）
- Linux 上改防火墙需要 ufw，并以管理员权限运行。未安装时，可在防火墙页面在线安装，或上传 deb/rpm 离线安装

## 快速开始

```bash
# 后端
go run ./cmd/server

# 前端（另开终端）
cd web && pnpm install && pnpm dev
```

- 页面：<http://127.0.0.1:5173>（`/api` 代理到 `:8080`）
- 只跑后端时：<http://127.0.0.1:8080>（使用已嵌入的页面）
- 首次启动管理员：`admin` / `admin123`

## 构建

```bash
make build
./bin/agentdeck
```

`make build` 会先构建前端并嵌进可执行文件。交叉编译前先执行 `make web`：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/agentdeck ./cmd/server
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/agentdeck.exe ./cmd/server
```

## 说明

- 防火墙的关闭、删除和重置会立刻作用到本机，操作前面板里会再确认一次。
- Linux 没有 ufw 时，防火墙页面会给出在线或离线安装，装好之后才能改规则。
- 字段一律 `snake_case`。成功时响应 `code` 为 `0`。

## 许可证

本项目基于 [Apache License 2.0](./LICENSE) 开源。
