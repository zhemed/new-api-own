# 证据：release.yml 附带 Linux 二进制附件

任务 `10-06-release-linux-binaries`。仅改 `.github/workflows/release.yml`，未 push、未触发工作流。

## 1. 关键发现（决定了实现方式）

### 1.1 前端是**嵌入**的，必须先构建

```
main.go:42: //go:embed web/dist
main.go:45: //go:embed web/dist/index.html
```
`web/dist` 缺失 → 构建失败或产出无界面二进制。所以**不能**只跑 `go build`。

### 1.2 版本号来源：镜像用的是 **tag**，不是仓库 `VERSION`

`Dockerfile:29`：`-X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'`
而 `docker-build.yml` 在构建前执行 `echo "${TAG}" > VERSION`。
→ 镜像里 `common.Version` = `v0.0.6`（仓库里 `VERSION` 文件本身是 `0.0.6`，不带 `v`）。
本工作流必须**同样先写 `VERSION`**，否则 Release 二进制会报 `0.0.6` 而镜像报 `v0.0.6`。

### 1.3 从 git 历史恢复了被删的旧 `release.yml`

```
4162f3d chore: 只保留镜像交付——移除 Electron 桌面壳、二进制 Release、GitCode 同步与分支镜像工作流
  D .github/workflows/release.yml
```

旧实现（`git show 4162f3d^:.github/workflows/release.yml`）→ 正好解释了我们线上看到的资产名：

| 旧做法 | 问题 |
|---|---|
| `go build -o new-api-$VERSION` / `-o new-api-arm64-$VERSION` | **带版本号**，面板无法按固定名拼 URL |
| `sha256sum new-api-*` → `checksums-linux.txt` | 名单名带 `-linux`，非 `SHA256SUMS` |
| amd64 用默认 CGO；arm64 用 `gcc-aarch64-linux-gnu` + `CGO_ENABLED=1` | 与镜像的 `CGO_ENABLED=0` **不一致** → 与镜像不同源 |
| `on: workflow_dispatch`（只有 `name` 输入，须在 tag ref 上跑） | 推 tag 不产 Release → 这次 v0.0.5/v0.0.6 断档的直接原因 |
| `softprops/action-gh-release`、`oven-sh/setup-bun` | 第三方 action |

线上 `v0.0.5` 的资产名（`new-api-v0.0.5` / `new-api-arm64-v0.0.5` / `checksums-linux.txt`）
与该旧实现**逐字吻合**，交叉验证成立。

## 2. 实现

### 2.1 附件契约（固定名，不含版本）

```bash
ASSETS="dist/new-api-linux-amd64 dist/new-api-linux-arm64 dist/SHA256SUMS"
```

### 2.2 与镜像同源：**用 Dockerfile 自己的工具链镜像**

不引入 `setup-go` / `setup-bun`（后者是第三方 action），而是直接用 `Dockerfile` 里那两个镜像，
并复用同一份构建命令：

| 环节 | 做法 | 与镜像的一致性 |
|---|---|---|
| 前端 | `docker run … oven/bun:1@sha256:0733e50…`，`bun install --frozen-lockfile` + `DISABLE_ESLINT_PLUGIN=true VITE_REACT_APP_VERSION="$(cat /build/VERSION)" bun run build` | 镜像 digest 与 `Dockerfile:2` **完全相同** |
| 后端 | `docker run … golang:1.25.1-alpine`，`CGO_ENABLED=0 GOWORK=off GOEXPERIMENT=greenteagc`，`-ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$VER"` | 与 `Dockerfile:11,12,17,29` 一致 |
| 交叉编译 | `GOOS=linux GOARCH={amd64,arm64}` | `CGO_ENABLED=0` 纯 Go，**无需 cross-gcc**（旧实现需要，因为它用了 CGO） |

### 2.3 保留的既有行为（全部未回退）

触发条件与 `docker-build.yml` 逐字一致；`workflow_dispatch`（`tag` 必填、复用同一脚本、非法拒绝）；
`gh release view` → `edit` 幂等；仅最高版本 `--latest`；带 `-` → `--prerelease` 且永不 latest；
`--verify-tag`；仅 `actions/checkout` 一个 action；并行风险注释保留。

### 2.4 幂等与自检

- 重跑：notes 刷新 + `gh release upload --clobber` 覆盖附件（不因 "asset already exists" 失败）；
- 上传后**断言三个固定附件都在**，缺失即 `::error::` 失败；
- 用**白名单**（而非"版本号特征正则"）检测多余附件——旧实现那种 `new-api-arm64-v0.0.5`
  用 `-<digits>.<digits>` 正则会漏掉，白名单不会漏；多余附件只 `::warning::`，
  因为旧 `v0.0.5` 的历史资产按用户要求不动。

## 3. 校验输出

```
=== YAML / triggers ===
release.yml parse OK ; docker-build.yml parse OK
triggers            : ['push', 'workflow_dispatch']
tags == docker-build: True ['v[0-9]*', '[0-9]*']
dispatch            : required=True type=string
permissions         : {'contents': 'write'}
steps               : ['Check out','Resolve and validate tag','Align VERSION with the tag',
                       'Build frontend','Build Linux binaries (amd64 + arm64)',
                       'Generate SHA256SUMS','Publish release']

=== no third-party actions ===
actions used        : ['actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0']
third-party?        : none

=== build parity with Dockerfile ===
bun  image  : df=oven/bun:1@sha256:0733e50…  rel=同  match=True
go   image  : df=golang:1.25.1-alpine        rel=同  match=True
flag CGO_ENABLED=0           df=True rel=True
flag GOWORK=off              df=True rel=True
flag GOEXPERIMENT=greenteagc df=True rel=True
ldflags pkg path    : df=True rel=True
frontend embedded   : main.go embed=True  (so frontend build is mandatory)

=== asset-name contract ===
declared assets     : ['dist/new-api-linux-amd64', 'dist/new-api-linux-arm64', 'dist/SHA256SUMS']
exact 3 fixed names : True
version in any name : none
only Linux (no mac/win): True
overwrite on re-run : --clobber True
```

```
=== legacy names: comment-only or executable? ===
  (no EXECUTABLE lines above = legacy names appear only in comments)

=== bash -n syntax check of every run: script ===
  PASS Resolve and validate tag      PASS Align VERSION with the tag
  PASS Build frontend                PASS Build Linux binaries (amd64 + arm64)
  PASS Generate SHA256SUMS           PASS Publish release
ALL SCRIPTS: PASS

=== branch matrix (real extracted decision block, real repo tags; HIGHEST=v0.0.6) ===
  TAG=v0.0.6           -> --latest         <none>
  TAG=0.0.6            -> --latest         <none>
  TAG=v0.0.5           -> --latest=false   <none>
  TAG=v0.0.7-alpha.1   -> --latest=false   --prerelease

=== illegal tag still rejected ===
::error::refusing to publish 'release-notes' …not a version tag…
exit=1

=== project gate ===
✅ 部署方式唯一：docker run + ghcr.io/zhemed/new-api-own
```

## 4. 与 `docker-build.yml` 的关系（一句话）

**同源同版本**：两者都由**同一 tag** 触发，用**同一组工具链镜像**、**同一组编译 flag**
（`CGO_ENABLED=0` / `GOWORK=off` / `GOEXPERIMENT=greenteagc`）与**同一个 `VERSION` 值**（tag 写入）构建；
镜像仍**唯一部署方式**，Release 上的二进制只是**更新器的输入**。

## 5. 注意：工作流改过名

`name:` 由 `Publish release (image-only)` → **`Publish release (image + Linux updater binaries)`**。
之前给的补发命令要跟着改：

```
gh workflow run "Publish release (image + Linux updater binaries)" -f tag=v0.0.6
```

## 6. 未做（等用户定）

- 旧 `v0.0.5` Release 上的 `new-api-v0.0.5` / `new-api-arm64-v0.0.5` / `checksums-linux.txt` **不动**；
- 存量 `v0.0.6` 的**实际**补发（需先落到默认分支 + 用户同意推送）；
- 未在真机跑过构建（禁止触发工作流）；构建步骤的**参数**已与 Dockerfile 逐项比对，
  但 `go build` 的实际产物需在首次真实运行时确认。
