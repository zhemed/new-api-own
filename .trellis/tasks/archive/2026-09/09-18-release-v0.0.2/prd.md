# 发布 v0.0.2：中性化部署注释、版本号推进、tag 与双标签镜像

## Goal

1) docker-compose 注释去掉内网 IP；2) `VERSION` 推进到 0.0.2 并建立 0.0.3 / 0.0.4 递增发版约定；
3) 打 tag 触发 CI 产出 GitHub Release；4) 镜像同时支持 `v0.0.2` 与 `0.0.2` 两种拉取标签。

## Requirements

1. `docker-compose.yml` 顶部注释里的 `线上主机` 改为中性描述（公开仓库不带内网地址）。
2. `VERSION` 由 `0.0.1` 推进到 `0.0.2`；发版约定为每次维护递增第三位（0.0.3、0.0.4 …），
   `VERSION` 不带 `v`、tag 带 `v`，两者版本一致。
3. `docker-build.yml` 对同一镜像同时发布 `<tag>` 与去掉 `v` 的等值标签，并同步多架构清单与 cosign 签名。
4. 打注释 tag `v0.0.2` 并推送，触发 `release.yml` 生成 GitHub Release 与 `docker-build.yml` 发布镜像。
5. 排障中发现并修复：`release.yml` / `electron-build.yml` 的 `-ldflags -X` 用了简写路径
   `new-api/common.Version`，Go 会静默忽略，导致 Release 二进制版本恒为 `v0.0.0`；改为完整模块路径
   `github.com/QuantumNous/new-api/common.Version`。
6. `MAINTENANCE.md` 增加发版流程章节（含双标签与 ldflags 注意事项）。

## Acceptance Criteria

- [x] `docker-compose.yml` 不再包含 `线上主机`
- [x] `VERSION` = `0.0.2`，发版约定写入 `MAINTENANCE.md`
- [x] 本地实测：完整模块路径可注入版本、简写路径输出 `v0.0.0`（验证结论写进 `MAINTENANCE.md`）
- [x] `v0.0.2` tag 已推送，`Release (Linux, macOS, Windows)` 运行成功
- [x] GitHub Release `v0.0.2` 含 `new-api-v0.0.2`、`new-api-arm64-v0.0.2`、`new-api-macos-v0.0.2`、
      `new-api-v0.0.2.exe` 与三个 checksums
- [x] 下载的 Release 二进制执行 `--version` 输出 `v0.0.2`
- [x] `Publish Docker image (Multi-arch)` 成功；GHCR 同时存在 `v0.0.2` 与 `0.0.2`（及 `latest`）
- [x] `docker run --rm ghcr.io/zhemed/new-api-own:{v0.0.2,0.0.2} --version` 均输出 `v0.0.2`

## Notes

- 上游历史已被压缩为自维护基线，仓库此前无任何 tag，因此 `Release` 工作流从未运行过 —— 这就是
  仓库没有 Releases 的原因。
- `Sync Release to GitCode` 工作流随 tag 触发但按条件跳过（仓库未设置 `GITCODE_REPOSITORY` 变量）。
- 下一次发版：改 `VERSION` 为 `0.0.3` → 提交 → `git tag -a v0.0.3` → 推送 `main` 与 tag。
