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

### 方式一：Docker 镜像（推荐，无需源码）

## 环境要求

- Docker **29.7.2** + Docker Compose **v5.4.0**（项目标准版本）
- 部署前运行 `./check-docker-env.sh` 校验环境

```bash
# 私有镜像，需要先登录 GHCR（使用 GitHub 凭据）
docker login ghcr.io -u zhemed

docker run -d --name new-api --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- 默认使用 SQLite，数据保存在 `./data` 目录
- 部署完成后访问 `http://localhost:3000`

### 方式二：源码构建（需要仓库访问权限）

```bash
# 私有仓库，clone 需要 GitHub 凭据
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# 单容器方式：构建并运行
docker build -t new-api-own .
docker run -d --name new-api --restart always \
  -p 3000:3000 -v ./data:/data new-api-own
```

### 生产部署参考

仓库内的 `docker-compose.yml` 是 host 网络模式 + PostgreSQL/Redis 的生产配置，按需调整默认密码后使用：

```bash
docker-compose up -d
```

## 维护

本项目由 [zhemed](https://github.com/zhemed) 维护。
