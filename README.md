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

> **唯一部署方式（强制）**：`docker run` + 公开镜像，单容器、host 网络、数据挂 `./data`。
> 不提供也不接受第二种形态（compose / Helm / K8s）；`scripts/forbid-extra-deploy-methods.sh`
> 在提交与 CI 上拦截，规则见 [.trellis/spec/guides/deployment-single-method.md](./.trellis/spec/guides/deployment-single-method.md)。

### 环境要求

- Docker **29.7.2**（项目标准版本）

一键安装：

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash
```

### 部署命令（无需源码）

```bash
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

- 默认使用 SQLite，数据保存在 `./data` 目录
- 部署完成后访问 `http://localhost:3000`
- 自建镜像时（需要仓库访问权限）先 `docker build -t new-api-own .`，再把上面的镜像名换成
  `new-api-own` —— **命令形态不变**，这不是第二种部署方式

可选环境变量（按需追加 `-e`）：限流开关、缓存、日志承载方式等见 [`.env.example`](./.env.example)
与 [MAINTENANCE.md](./MAINTENANCE.md)。

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

推 tag **只自动触发镜像构建**：向 `ghcr.io/zhemed/new-api-own` 发布 `v<版本>`、`<版本>` 与 `:latest`
（多架构清单 + cosign 签名）。

**三平台二进制 / Electron / GitCode 同步已改为手动触发**（避免每次发版扇出多个工作流）：

```bash
# Release（必须在 tag ref 上运行，否则上传步骤会被 if: refs/tags/ 跳过）
gh workflow run release.yml --ref v0.0.3
# Electron 打包 / GitCode 同步同理，按需手动运行
gh workflow run electron-build.yml
gh workflow run sync-release-to-gitcode.yml -f tag_name=v0.0.3
```
