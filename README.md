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

### 升级 / 回滚（照做不会停在旧版）

> **关键认知：重启容器不会换镜像。** `docker restart`（或 `--restart always` 自动重启）只是把
> **同一个镜像**再跑一遍，版本当然不变。升级必须**删掉容器、用新镜像重建**。

升级三步（`<版本>` 换成目标 tag，如 `0.0.6`；**在原来的部署目录执行**，`./data` 是相对路径）：

```bash
docker pull ghcr.io/zhemed/new-api-own:<版本>   # 1. 先拉新镜像
docker rm -f new-api                            # 2. 删旧容器（数据在 ./data，不受影响）
docker run -d --name new-api --restart always \
  --network host -v ./data:/data \
  ghcr.io/zhemed/new-api-own:<版本>             # 3. 用新镜像重建
```

回滚同理，只把 tag 换成上一个版本（registry 保留每个历史 tag）：

```bash
docker pull ghcr.io/zhemed/new-api-own:0.0.5
docker rm -f new-api
docker run -d --name new-api --restart always --network host -v ./data:/data \
  ghcr.io/zhemed/new-api-own:0.0.5
```

**为什么显式 `docker pull`、为什么不建议用 `:latest`**：`:latest` 是**本地标签**——
`docker run …:latest` 不会去远端比对更新，本地有就直接用；本地标签陈旧时就会"升级了却还是旧版"
（2026-10-06 本机 `latest` 曾停在 `0.0.3`，当日已刷新；**任何机器的本地标签都可能陈旧**）。所以升级用
**固定 tag**（`0.0.6` 与 `v0.0.6` 指向同一份多架构清单）；非要用 `:latest`，必须先
`docker pull ghcr.io/zhemed/new-api-own:latest` 再重建。

> 判断本机镜像是否已刷新：`docker images ghcr.io/zhemed/new-api-own --digests`。
> **过滤必须带完整仓库名**——`docker images | grep 0.0.5` 会命中别的项目的同号 tag，
> 而且 `grep` 里 `.` 是通配符，会给出错误结论。

升级后确认版本（实例自身报告的才是真相）：

```bash
curl -s http://127.0.0.1:3000/api/status | grep -o '"version":"[^"]*"'
# 预期："version":"v0.0.6"；对不上就按 MAINTENANCE.md「版本与升级（每次维护必做）」逐处比对
# 有本仓库源码时，一条命令核对四项（仓库 VERSION / 最新 tag / registry 摘要 / 实例版本）：
#   make check-version INSTANCE=http://127.0.0.1:3000
```

> 升级 / 回滚**同样只有 `docker run` 这一种形态**：不要为此引入 compose / Helm / K8s / systemd
> （闸门 `scripts/forbid-extra-deploy-methods.sh` 会拦）。

### 面板内更新（临时跟上版本，不是第二种部署方式）

面板 **系统设置 → 系统维护 → 检查更新** 会显示当前版本与最新版本；有新版本时可直接点「立即更新」：
后端下载**本机架构**的 Linux 二进制 → 用发布的 `SHA256SUMS` 校验 → 校验通过才原子替换并原地重执行
（**校验失败绝不替换**）。检查与下载都发生在服务端，浏览器不直连 GitHub。

> ⚠️ **它是临时手段**：容器内被替换的二进制**不在镜像里**，下一次 `docker rm` + `docker run`
> 会退回镜像内版本。要长期稳定在新版本，仍然走上面的镜像三步。
> 开关：「检查更新」默认**关闭**（`UPDATE_CHECK_ENABLED=true` 开启），「立即更新」默认**允许**
> （`UPDATE_APPLY_ENABLED=false` 可整体关掉）；GitHub 直连不通时用 `HTTPS_PROXY`，或用
> `UPDATE_CHECK_API_BASE_URL` 指向镜像/代理。细节、排障与回退见
> [MAINTENANCE.md](./MAINTENANCE.md)「面板内更新（self-update）」。

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

**交付只有镜像这一条路径**：推 tag 触发 `Publish Docker image (Multi-arch)`，产出
`v<版本>` / `<版本>` / `:latest`。二进制 Release、Electron 桌面壳、GitCode 同步均已移除。
