# New API

自用 LLM 网关与 AI 资产管理平台：多模型聚合、Key 管理、计费与分发。

## 特性

- **多模型聚合**：OpenAI、Claude、Gemini、DeepSeek、Qwen 等主流模型统一接入
- **API 格式转换**：OpenAI 兼容 ⇄ Claude Messages、OpenAI → Gemini 等格式互转
- **Key 管理**：多 Key 汇聚、分组、模型限制、用量统计与可视化看板
- **计费与配额**：按请求、用量、缓存命中计费，灵活的计费策略
- **授权登录**：Discord、Telegram、LinuxDO、OIDC 统一认证
- **智能路由**：加权随机、失败自动重试、用户级模型限流
- **思考模式支持**：OpenAI o 系列、Claude thinking、Gemini thinking
- **现代化 UI**：简洁美观的界面与多语言支持

## 部署

### 环境要求

- Docker **29.7.2** + Docker Compose **v5.4.0**（项目标准版本）

一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash
```

### 方式一：一条命令部署（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-compose.sh | sudo bash
```

装到 `/opt/docker/new-api-own`。脚本会：生成随机数据库口令并写入 `.env`（600；重跑**沿用**既有口令，
绝不重新生成）→ 拉取同 ref 的 `docker-compose.yml` → 启动 new-api + PostgreSQL + Redis →
等到健康检查通过 → 收紧 `data/`、`logs/` 权限。镜像 tag 取自该 ref 上的 `VERSION`，
要钉版本用 `--ref v0.0.3`。

已有部署默认**拒绝覆盖**，`--force` 才重写并先备份；`./data`、`./logs` 与 `pg_data` 卷永不被删除。
参数见 `--help`（`--dir` 换目录、`--project-name` 换项目名、`--name-prefix` 让容器名共存、`--no-start` 干跑）。

> 部署前请读 [MAINTENANCE.md](./MAINTENANCE.md) 的「部署安全基线」：默认 host 网络会把面板暴露在
> 所有网卡上（含公网），且 `/api/status` 无需认证即可读。

### 方式二：Docker 镜像（单容器，无需源码）

```bash
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- 默认使用 SQLite，数据保存在 `./data` 目录
- 部署完成后访问 `http://localhost:3000`

### 方式三：源码构建（需要仓库访问权限）

```bash
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# 单容器方式：构建并运行
docker build -t new-api-own .
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data new-api-own
```

### 生产部署参考

仓库内的 `docker-compose.yml` 是 host 网络模式 + PostgreSQL/Redis 的生产配置。其中的变量都带默认值，
**单独 clone 后不建 `.env` 也能直接起**（默认口令 `123456`，仅供试跑）。正式部署请用「方式一」，
它会把随机口令写进 `.env`；手工部署则自己在同目录建 `.env`：

```bash
cat > .env <<'EOF'
POSTGRES_PASSWORD=<换成随机值>
REDIS_PASSWORD=<换成随机值>
EOF
docker compose up -d
```

## 维护

本项目由 [zhemed](https://github.com/zhemed) 维护，使用 [Trellis](https://github.com/mindfold-ai/trellis) 任务流程与 GitHub Actions 自动化发版。完整手册见 [MAINTENANCE.md](./MAINTENANCE.md)。

接入（工作流产物与技能随仓库分发，每台机器初始化一次身份即可）：

```bash
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own
trellis init --dsh -u <你的开发名> -s -y   # 生成 00-join-<开发名> 接入任务，不覆盖仓库文件
```

发版：把 `VERSION` 递增（0.0.3、0.0.4 …）提交后打 tag：

```bash
git tag -a v0.0.3 -m "v0.0.3"
git push origin main v0.0.3
```

CI 会生成 GitHub Release（Linux/macOS/Windows 二进制 + checksums），并向 `ghcr.io/zhemed/new-api-own` 发布 `v0.0.3` 与 `0.0.3` 双标签镜像（多架构 + cosign 签名）。
