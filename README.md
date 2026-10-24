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
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:latest
```

- 默认使用 SQLite，数据保存在 `./data` 目录。**该目录必须位于本仓库工作区内**（不要挂工作区外的绝对路径）：
  `.gitignore:27` 已忽略 `/data/`，数据不入库；迁目录 / 换挂载见下面[数据目录（必须在工作区内）](#数据目录必须在工作区内)
- 上面四条 `LOG_*` 就是**当前在用的日志模式**：日志放内存（体积上限 **200MB**、保留 **7 天**、
  写入时按字节裁剪，**没有定时清理任务**）——省磁盘写入，代价是**日志重启即丢**；
  想回到落盘模式，去掉 `LOG_SQL_DSN=memory` 重建即可。详见 [MAINTENANCE.md](./MAINTENANCE.md)
- `--log-opt` 限制的是 **Docker 自己的 json 容器日志**（另一个写入方，默认不轮转、会一直长）：
  每个文件 10MB、最多 3 个（≈30MB 上限）。它与应用自己的 `/data/logs/*.log` 是两回事
- 部署完成后访问 `http://localhost:3000`
- 自建镜像时（需要仓库访问权限）先 `docker build -t new-api-own .`，再把上面的镜像名换成
  `new-api-own` —— **命令形态不变**，这不是第二种部署方式

### 数据目录（必须在工作区内）

`-v ./data:/data` 里的 `./data` 就是实例数据目录（SQLite 库、日志、上传文件），**必须位于本仓库工作区内**：

- `.gitignore:27` 的 `/data/` 只匹配仓库根下的 `data/`——数据（凭据、用户与用量、日志）因此**不会进 Git**；
  挂到工作区外则不受这条规则保护，也不随工作区备份走。
- 维护会话（含 AI agent）的可读写范围**只有工作区**：数据在工作区内，文件数、sha256、SQLite 计数这些
  只读盘点可以直接做；放到区外，同一次盘点就成了越界读取，只能靠人工在主机上做。

换数据目录（或从别处迁入）按这个顺序，别跳步：

```bash
docker stop new-api                                # 1. 先停容器，别删（留着 docker inspect 对照参数）
mkdir -p ./data && cp -a <旧数据目录>/. ./data/     # 2. 复制，不要 mv（旧目录先当回滚点）
( cd <旧数据目录> && find . -type f | wc -l ) ; ( cd ./data && find . -type f | wc -l )   # 3. 文件数须相等
( cd <旧数据目录> && find . -type f -exec sha256sum {} + | sort | sha256sum )
( cd ./data        && find . -type f -exec sha256sum {} + | sort | sha256sum )            # 3. 两条摘要须一致
docker rm new-api                                  # 4. 删容器：不要带 -v
# 5. 照抄上面「部署命令（无需源码）」原样重建，只把 -v 的源改成 ./data
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status   # 6. 期望 200
```

- 第 3 步的 `find` 必须在**各自数据目录内部**执行（上面的子 shell `cd` 就是为此）：在仓库根直接跑
  `find . -type f` 会把整个仓库算进去，文件数与 sha256 摘要都不对。
- **必须先停容器再复制**：SQLite 的 `-wal` / `-shm` 在运行中与主库不一致，边跑边拷会拿到损坏的库。
- 确认新挂载真的生效：
  `docker inspect -f '{{range .Mounts}}{{.Source}}{{"\n"}}{{end}}' new-api` 的输出应等于当前目录下的 `data`。
- 旧目录是回滚点：等实例 200、版本与业务计数（用户 / 渠道 / 日志条数）都对上再删——
  删掉之后 `./data` 就是唯一副本，误挂载或误删都没有第二条退路。

### 升级 / 回滚（照做不会停在旧版）

> **关键认知：重启容器不会换镜像。** `docker restart`（或 `--restart always` 自动重启）只是把
> **同一个镜像**再跑一遍，版本当然不变。升级必须**删掉容器、用新镜像重建**。

升级三步（`<版本>` 换成目标 tag，如 `0.0.48`；**在原来的部署目录执行**，`./data` 是相对路径）：

```bash
docker pull ghcr.io/zhemed/new-api-own:<版本>   # 1. 先拉新镜像
docker rm -f new-api                            # 2. 删旧容器（数据在 ./data，不受影响）
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:<版本>             # 3. 用新镜像重建
```

回滚同理，只把 tag 换成上一个版本（registry 保留每个历史 tag）：

```bash
docker pull ghcr.io/zhemed/new-api-own:0.0.47
docker rm -f new-api
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:0.0.47
```

> **回滚注意**：上面四条 `LOG_*` 需要支持内存日志的版本（`LOG_SQL_DSN=memory` 自 `v0.0.4` 起，
> `LOG_MEMORY_MAX_BYTES` 自 `v0.0.8` 起）。回滚到更早版本时，请把该版本不认识的变量去掉再重建。

> **受影响版本（v0.0.43–v0.0.45）**：这几个版本的资产白名单漏了签名资产 `SHA256SUMS.sig`，
> **面板内更新会稳定失败**，报「更新包的签名与公布的校验值不匹配，因此已拒绝替换。」
> —— **发布资产本身没有问题，不是被篡改**。处置：按上面的镜像三步升级一次
> （`docker pull` 新 tag → `docker rm -f new-api` → 按原参数重建）；升到 **v0.0.46 或更新版本**后面板自更新恢复。


**为什么显式 `docker pull`、为什么不建议用 `:latest`**：`:latest` 是**本地标签**——
`docker run …:latest` 不会去远端比对更新，本地有就直接用；本地标签陈旧时就会"升级了却还是旧版"
（2026-10-06 本机 `latest` 曾停在 `0.0.3`，当日已刷新；**任何机器的本地标签都可能陈旧**）。所以升级用
**固定 tag**（`0.0.48` 与 `v0.0.48` 指向同一份多架构清单）；非要用 `:latest`，必须先
`docker pull ghcr.io/zhemed/new-api-own:latest` 再重建。

> 判断本机镜像是否已刷新：`docker images ghcr.io/zhemed/new-api-own --digests`。
> **过滤必须带完整仓库名**——`docker images | grep 0.0.48` 会命中别的项目的同号 tag，
> 而且 `grep` 里 `.` 是通配符，会给出错误结论。

升级后确认版本（实例自身报告的才是真相）：

```bash
curl -s http://127.0.0.1:3000/api/status | grep -o '"version":"[^"]*"'
# 预期："version":"v0.0.48"；对不上就按 MAINTENANCE.md「版本与升级（每次维护必做）」逐处比对
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
> 受影响版本（v0.0.43–v0.0.45）会稳定失败并报签名校验不匹配：先按上面「升级 / 回滚」的提示用镜像方式升级一次即可恢复。

可选环境变量（按需追加 `-e`）：限流开关、缓存、日志承载方式等见 [`.env.example`](./.env.example)
与 [MAINTENANCE.md](./MAINTENANCE.md)。

> 想按**体积**控内存：加 `-e LOG_MEMORY_MAX_BYTES=200MB`——写入触发（不必等定时器）、超预算自动裁最老的记录；
> 详见 [MAINTENANCE.md](./MAINTENANCE.md)「弱盘机器：把日志放到内存（或独立盘）」。

## 维护

本项目由 [zhemed](https://github.com/zhemed) 维护，使用 [Trellis](https://github.com/mindfold-ai/trellis) 任务流程与 GitHub Actions 自动化发版。完整手册见 [MAINTENANCE.md](./MAINTENANCE.md)。

接入（工作流产物与技能随仓库分发，每台机器初始化一次身份即可）：

```bash
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own
trellis init --dsh -u <你的开发名> -s -y   # 生成 00-join-<开发名> 接入任务，不覆盖仓库文件
```

发版：把 `VERSION` 递增（0.0.48 → 0.0.49 …）提交后打 tag：

```bash
git tag -a v0.0.49 -m "v0.0.49"
git push origin main v0.0.49
```

推 tag **只自动触发镜像构建**：向 `ghcr.io/zhemed/new-api-own` 发布 `v<版本>`、`<版本>` 与 `:latest`
（多架构清单 + cosign 签名）。

**交付只有镜像这一条路径**：推 tag 触发 `Publish Docker image (Multi-arch)`，产出
`v<版本>` / `<版本>` / `:latest`。二进制 Release、Electron 桌面壳、GitCode 同步均已移除。
