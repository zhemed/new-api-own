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

## 快速开始

```bash
# 克隆项目
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own

# 编辑 docker-compose.yml（默认使用 ghcr.io/zhemed/new-api-own:latest 镜像）
nano docker-compose.yml

# 启动服务
docker-compose up -d
```

部署完成后访问 `http://localhost:3000` 即可开始使用。

## 数据

- 默认使用 SQLite，数据保存在 `./data` 目录
- 数据目录可挂载到任意绝对路径，便于备份迁移

## 维护

本项目由 [zhemed](https://github.com/zhemed) 维护。
