# 调查版本呈现链（复用编制）

## Goal

把「用户能看到版本号」的所有位置找全并逐个实测（本机 0.0.6 环境），回答：
**哪个位置会让用户看到 0.5 或旧版本号**。只读调查，不做任何修复。

## Requirements

- 找全前端版本注入与消费点：`VITE_REACT_APP_VERSION`（`Dockerfile:9`、`makefile:13`）→
  `web/src/lib/build-metadata.ts:68` → 页脚/关于/系统信息/设置页等 UI 出口。
- 找全后端版本链：`VERSION` 文件 → `Dockerfile:29` 的 `-ldflags -X common.Version` →
  `common/constants.go:14` 默认值 `v0.0.0` → `common/init.go:37` 环境变量覆盖 → `/api/status` 响应字段。
- 镜像元数据：`docker image inspect ghcr.io/zhemed/new-api-own:latest` 的 OCI labels、
  `Config.Env`、镜像内 `/VERSION` 文件、容器内实际文件。
- 运行时实测本机 3020 演示实例（0.0.6）：用 `bw` 打开，记录页面显示的版本、`/api/status` 返回值、
  加载的前端资源哈希，并与工作区 `web/dist` 产物哈希对比。
- **重点**：检查浏览器侧是否存在缓存 / Service Worker / CDN 缓存导致页面停在旧构建；贴资源哈希证据。
- 结论必须明确：哪些位置正常显示 0.0.6、哪些位置会显示 0.5 或旧版本（含推断依据）。

## Acceptance Criteria

- [ ] 版本呈现位置清单（文件:行），覆盖前端注入/消费、后端 ldflags/env/API、镜像元数据、运行时页面。
- [ ] 每处实测结果（贴命令输出或页面证据）。
- [ ] 浏览器缓存/Service Worker 排查结论 + 资源哈希对比证据。
- [ ] 「会显示 0.5 或旧版本」的结论首选并给出依据；无法验证的项单独列出。
- [ ] 给 Lead 的 ≤25 行中文汇报。

## Constraints

- **只读**：不改任何文件（含 `web/**`、`.trellis/**` 除本任务目录外）；不修复。
- 无外部网络；不动用户远程实例；不停/删容器与进程；不 commit/push；禁止 `git add -A` 与 worktree。
- 允许本机 `bw` 访问 3020 演示实例（Lead 本轮点名）。
