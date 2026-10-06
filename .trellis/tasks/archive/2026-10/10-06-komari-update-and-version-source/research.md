# komari 更新机制调研 + 版本源修复

任务 `10-06-komari-update-and-version-source`。只读外部（github.com/zhemed/komari、api.github.com、ghcr），未 push、未触发工作流。

---

## 1. 根因（**已用线上 API 直接证实**）

面板的"检查更新"在前端发起（`web/src/features/system-settings/maintenance/update-checker-section.tsx:63-71`）：

```ts
const response = await fetch(
  'https://api.github.com/repos/zhemed/new-api-own/releases/latest',
  { headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'new-api-dashboard' } }
)
```

它读的是 **`/releases/latest`**，由 **Release 条目**驱动，与镜像 tag 无关。实测该端点：

```
GET https://api.github.com/repos/zhemed/new-api-own/releases/latest
→ "tag_name": "v0.0.5", "draft": false, "prerelease": false, "body": null
   assets: checksums-linux.txt (168B)
           new-api-arm64-v0.0.5  (115 MB)
           new-api-v0.0.5        (118 MB)
```

- `v0.0.6` **没有任何 Release 条目** → `/releases/latest` 仍停在 `v0.0.5` → 面板报"有新版本可用：v0.0.5"。
- 附带发现：**旧 Release 工作流是带二进制的**（118MB 可执行文件 + 校验和）。
  这本身就是"第二种部署形态"，与 `.trellis/spec/guides/deployment-single-method.md`
  （唯一部署方式 = `docker run` + 公开镜像）冲突。所以恢复 Release **不能**照搬旧形态。

---

## 2. komari 更新机制（文件:行 级证据）

> 行号按 `main` 分支当前内容计（`raw.githubusercontent.com` 抓取，未 clone）。规范文档
> `.trellis/spec/backend/server-upgrade.md` 自述"锚点会随代码漂移"，引用时请连同锚点文本一起核。

### 2.1 更新源与抓取 `internal/upgrade/releases.go`

| 项 | 实现 | 位置 |
|---|---|---|
| 源 | **GitHub Releases REST API** | `DefaultAPIBase = "https://api.github.com"` `releases.go:22` |
| 端点 | `GET {base}/repos/{owner}/{repo}/releases?per_page=100` | `releases.go:113` |
| 鉴权 | 可选 `Authorization: Bearer <token>`（私有仓库/限流用）；**`User-Agent` 必填**，否则 GitHub 拒绝 | `releases.go:26`、`releases.go:91-95` |
| 头 | `Accept: application/vnd.github+json` | `releases.go:92` |
| 超时 | `http.Client{Timeout: 30 * time.Second}` | `releases.go:69` |
| 限流/重试 | **无重试、无退避、无 ETag/缓存**；非 200 直接返回 `github api %s: http %d` | `releases.go:101-102` |
| draft | 跳过 | `releases.go:119-121` |
| 目标仓库 | 只来自服务端设置 `server_update_repo`（默认 `zhemed/komari`），**接口不接受请求方传入 URL** | 规范 §0/§2 |

### 2.2 版本比较 `releases.go:212-256`

- `ParseVersion`：`TrimSpace` → 去掉**单个**前导 `v` → 在首个 `-`/`+` 处**截断**（`0.0.7-rc1` → `0.0.7`）
  → 按 `.` 切 2~3 段、逐段 `Atoi` 且 `>=0` → 补零到 3 段。
- `CompareVersions`：逐段比较，返回 `-1/0/1`；任一侧解析失败返回 `ok=false`。
- **预发布不靠 semver 优先级，靠"过滤"**：`StableReleases` 直接跳过 `prerelease==true`（`releases.go:147-160`）。
- **排序**：按版本降序，同版本用 `published_at` 兜底（`releases.go:135-142`）。
- **降级策略**：`LatestStable` 取 `list[0]`；若版本号无法比较，**保守判定为"有新版本"**
  （只要 tag 不同），交用户决定（`releases.go:171-176`）。

### 2.3 是否真自更新——**是，四档模式**（`internal/upgrade/upgrade.go:91-105`）

| 模式 | 触发条件 | 行为 |
|---|---|---|
| `ModeBinary` | linux + systemd + 目录可写 | 下载 → 校验 → 自检 → **原子替换二进制** → 退出，systemd 拉起 |
| `ModeDockerRecreate` | 容器 + **挂了可用 docker socket** | 拉镜像 → **helper 容器重建自身**（`cmd/dockerSelfRecreate.go` + `internal/dockerapi`） |
| `ModeContainerReplace` | 容器 + 无 socket + 目录可写 | **在容器内替换二进制 + 原地重执行** |
| `ModeManual` | 容器 + 目录不可写（只读 rootfs） | 只给可复制的 `PullHint` 命令，**不做任何替换** |
| `ModeDownloadOnly` | 无 systemd 且非容器 | 只下载，不替换不退出 |

安全不变量：资产 `komari-SHA256SUMS` 流式 SHA256 校验，不符则删临时文件、**绝不替换**；
替换前 `VerifyBinary` 跑 `<新二进制> --help` 并要求输出含 `Komari Monitor <tag>`；
替换 = `rename(旧→backup)` + `rename(新→正式)`，失败尝试回滚（规范 §4）。

### 2.4 **对我们最关键的一条**：容器内替换二进制会被 recreate 丢掉

komari 自己把它写成了**显式代价**（`upgrade.go:96-99`）：

```go
// ModeContainerReplace：容器 + 没有可用 socket —— 在容器内替换二进制后原地重执行
// （唯一"零配置"的容器网页升级方式：不要求挂载、不要求 restart 策略）。
// 代价：**重建容器**会退回镜像里的版本（docker restart 不会）。
```

规范 §5 记载这还造成过一次**真实事故**：host 网络容器里"自身容器识别"依赖 hostname==短 ID
与 cgroup，host 网络下两条线索都失效 → 落回 `container-replace` → **"升级当次看起来成功，但镜像不变；
之后任何一次容器重建（`docker rm` + 同命令重跑）都会把版本退回镜像版本。"**

---

## 3. 可移植性结论（我们：`docker run` 单容器 + `--network host` + 未挂 socket）

| komari 做法 | 我们能用吗 | 理由 |
|---|---|---|
| **Releases 作更新源** | ✅ 已用 | 我们面板本来就读 `/releases/latest`，问题不是"源选错"而是"Releases 缺条目" |
| UA 必填 / Accept 头 | ✅ 已有 | 前端 `update-checker-section.tsx:66-69` 已带 |
| semver 比较（去 `v`、忽略后缀、2~3 段） | ✅ 已有 | 我们已有 `web/.../utils/version-compare.ts` 与 `common/version_compare.go` |
| draft 跳过 / **prerelease 不进 latest** | ✅ 已移植 | 已写入 `release.yml`：带 `-` 的 tag → `--prerelease`，GitHub 不再从 `/releases/latest` 返回它 |
| 限流处理 | ➖ 维持现状 | komari 自身也没做重试；我们是"提示"路径，非关键路径，不引入复杂度 |
| **二进制自更新**（下载/校验/原子替换/重启） | ❌ **不可用** | ① 容器内替换会被下一次 `rm`+`run` 抹掉（komari 自证 + 事故记录）；② 需要发布二进制资产 → 直接违反"唯一部署方式 = 镜像" |
| `docker-recreate`（挂 socket 重建自身） | ❌ 不适用 | 需挂 socket（我们刻意不挂）；且 `--network host` 下自身容器识别失效 |
| `--help` 自检代替 `--version` | ➖ 不适用 | 那是为它的 Go 二进制设计的；我们没有二进制交付面 |

**结论：只移植"检查 + 提示 + 指引"，绝不移植"自更新"。**
我们的升级路径仍然是 README 的三步（`docker pull` → `docker rm -f` → `docker run`），
这也与上一任务实测的结论一致（重启 ≠ 换镜像，容器按镜像 ID 固定）。

---

## 4. 修复：选 (a) —— 恢复**仅镜像**的 Release 工作流

新增 `.github/workflows/release.yml`。**选 (a) 而不选 (b) 的理由**：

1. 面板读的就是 `/releases/latest`，补 Release **零后端改动**即可修好；选 (b) 要改 Go/前端更新检查，
   与后端成员范围重叠、风险与工作量都更大。
2. (b) 要让面板改读 ghcr（`/token?scope=...` 取匿名 token + 分页 + 摘要解析），
   比"补一个 Release"复杂得多，还会与 komari 的成熟做法分叉。
3. (a) 能顺带恢复人类可读的发布说明与 `html_url`（面板"Open release"按钮依赖它）。

**与旧形态的关键差异：附件为零。** 旧 `v0.0.5` 带 118MB 二进制；新工作流**不附加任何资产**，
发布说明只给镜像拉取命令与升级三步，避免重新引入第二种部署形态。

关键设计点（全部已校验）：

| 点 | 做法 |
|---|---|
| 触发条件 | `tags: ['v[0-9]*', '[0-9]*']` —— 与 `docker-build.yml` **逐字一致**（已断言） |
| 不重复建 Release | 先 `gh release view`；已存在则 `gh release edit` 刷新说明（幂等） |
| 权限最小化 | 仅 `contents: write`（不需要 `packages`） |
| 第三方 action | **不引入**（不用 `softprops/action-gh-release`），用 runner 预装的 `gh` CLI |
| `latest` 防回退 | 仅"最高版本 tag"可拿 `--latest`；否则 `--latest=false`。避免重跑旧 tag 把 `latest` 拉回旧版（与已修的镜像 `latest-<arch>` 同类问题） |
| prerelease | 带 `-` 的 tag → `--prerelease`，且永不 `--latest` |
| 可验证 | 结尾 `gh release view --json tagName,isLatest,isPrerelease,assets` 写入 run summary |
| tag 校验 | `--verify-tag` |

---

## 5. 校验输出

```
release.yml YAML parse: OK
docker-build.yml YAML parse: OK

release.yml triggers      : ['push']
docker-build.yml triggers : ['push', 'workflow_dispatch']
tags equal & identical    : True ['v[0-9]*', '[0-9]*']

release.yml permissions   : {'contents': 'write'} (least privilege: contents:write only)
attaches binaries?        : False
uses 3rd-party action?    : False (uses preinstalled gh CLI instead)
idempotent re-run guard   : True
verify-tag used           : True
```

分支逻辑矩阵（`latest` / `prerelease` 决策）：

```
TAG=v0.0.6             HIGHEST=v0.0.6        -> --latest         <none>
TAG=v0.0.5             HIGHEST=v0.0.6        -> --latest=false   <none>
TAG=0.0.6              HIGHEST=0.0.6        -> --latest         <none>
TAG=v0.0.7-alpha.1     HIGHEST=v0.0.7-alpha.1 -> --latest=false --prerelease
TAG=v0.0.7-alpha.1     HIGHEST=v0.0.6        -> --latest=false   --prerelease
```

项目闸门：`bash scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一` exit 0。

---

## 6. 仍未闭环（需 Lead/用户决策）

1. **存量 `v0.0.6` 需要补一条 Release**：本工作流只在 **新 tag 推送**时触发，
   当前 registry 已有 `0.0.6` 镜像但没有 Release。补法：`git push origin v0.0.6`（重新推 tag 会触发），
   或手动 `gh workflow run` / `gh release create`。**需授权**（我未执行）。
2. **顺序风险**：`release.yml` 与 `docker-build.yml` 同 tag 并行触发。若镜像推送失败而 Release 成功，
   面板会提示一个拉不到的版本。彻底闭环需改成 `workflow_run` 链式（等镜像成功后再建 Release），
   本轮按 Lead 指定保持 `on: push: tags`，仅记录该风险。
3. 旧 `v0.0.5` Release 仍挂着 118MB 二进制资产（与"唯一部署方式=镜像"冲突），是否清理需用户决定。
