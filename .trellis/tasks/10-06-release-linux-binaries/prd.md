# release.yml 附带 Linux 二进制附件

## Goal

按用户范围变更：**面板要能直接更新**，故 `release.yml` 除 Release 条目外，还要挂 **Linux 二进制附件**，
命名固定、不带版本号，供面板更新器按名字拼 URL。

## Requirements

### R1 附件（新增）

| 资产名（**固定，不带版本号**） | 内容 |
|---|---|
| `new-api-linux-amd64` | linux/amd64 静态二进制 |
| `new-api-linux-arm64` | linux/arm64 静态二进制 |
| `SHA256SUMS` | `sha256  <文件名>` 清单，覆盖上面两个 |

- **只出 Linux**，不出 macOS / Windows。
- 命名**不得**带版本号（旧 `v0.0.5` 用的 `new-api-v0.0.5` / `new-api-arm64-v0.0.5` 是反面样本）。

### R2 与 `docker-build.yml` **同源同版本**

二进制必须与镜像用同一套构建方式，且版本号一致：

- `CGO_ENABLED=0` 静态；
- `GOEXPERIMENT=greenteagc`、`GOWORK=off`（与 Dockerfile / docker-build.yml 一致）；
- `-ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=<版本>'"`；
- 版本取值必须与镜像一致：`docker-build.yml` 是 `echo "${TAG}" > VERSION` 后再 build，
  故本工作流也必须**先把 tag 写入 `VERSION`** 再构建（而不是直接用仓库里不带 `v` 的 `VERSION`）。
- **前端必须先构建**：`web/dist` 会被 Go 侧 embed，缺失会导致构建失败或产出无界面二进制。

### R3 保留既有全部行为（不得回退）

- 触发条件与 `docker-build.yml` **逐字一致**；
- `workflow_dispatch`（`tag` 必填、复用同一脚本、只允许版本 tag、非法拒绝）；
- `gh release view` → `edit` **幂等**；
- 仅**最高版本** tag 可 `--latest`；带 `-` 的 → `--prerelease` 且**永不** latest；
- **不引入第三方 action**（用预装 `gh` CLI；`actions/checkout` 为已用的一方法定 action）。

### R4 幂等与附件

- 重跑同一 tag 时：Release notes 刷新 + 附件**覆盖上传**（`gh release upload --clobber`），
  不得因 "asset already exists" 失败。

## 校验要求

- YAML parse 通过；
- 分支矩阵（latest / prerelease）输出；
- **断言附件名固定、且不含带版本号的历史命名**；
- 说明与 `docker-build.yml` 的关系：镜像 + Release 二进制**同源同版本**。

## Constraints（硬约束）

- 写范围**仅** `.github/workflows/` + 本任务目录；
- **不 push、不触发任何工作流**；
- 不 commit；只读外部网络；
- 不碰生产机与本机 3020、不碰 `.local-instance/`；
- 旧 `v0.0.5` Release 上的历史命名二进制**不动**（等用户定）。

## Acceptance Criteria

- [ ] 三个附件名固定且不带版本号，有可复现的断言输出
- [ ] 构建参数与 `docker-build.yml`/Dockerfile 同源（`CGO_ENABLED=0`、`GOEXPERIMENT=greenteagc`、`GOWORK=off`、ldflags 路径一致）
- [ ] 版本号来源与镜像一致（tag 写入 `VERSION` 后再构建），前端已构建
- [ ] R3 全部行为保留（触发一致 / dispatch / 幂等 / latest / prerelease / 无第三方 action）
- [ ] 重跑时附件覆盖而非失败
- [ ] YAML parse + 矩阵 + 命名断言均有实测输出
- [ ] 未 push、未触发工作流、未动其它文件

## Notes

- 来源：Lead 指派（承接 `10-06-release-dispatch-backfill`）。
- 注意 AGENTS.md「唯一部署方式」：附件是**更新器的输入**，不构成第二种部署方式；
  仍需在注释中说明镜像才是唯一部署形态。
