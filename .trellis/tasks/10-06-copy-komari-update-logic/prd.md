# 抄 komari 的在线更新逻辑（团队）

## 用户要求（2026-10-06）

1. 生产机的"检查更新"显示 **"有新版本可用：v0.0.5"**，而实际最新是 **0.0.6** —— 更新源错了（面板读 GitHub Releases，而 Releases 停在 v0.0.5，因为 Release 工作流在瘦身时被删，之后只有镜像）；
2. 「抄一下 https://github.com/zhemed/komari 它的在线更新逻辑，看看能不能用得上，调用团队去做」。

## 授权边界

- **允许**访问 `github.com/zhemed/komari`（用户点名）及 `api.github.com`、`ghcr.io/zhemed/new-api-own`（只读）；
- 其余外部主机一律不访问；生产机不碰；不 commit/push（Lead 统一处理，用户当前要求"先放着不提交"）。

## 三条线（写范围互不重叠）

| 成员 | 任务 | 写范围 |
|---|---|---|
| `hunt-deploy-chain` | 读 komari 实现 → 给出可移植设计 + 修好**版本源**（Release 条目 或 registry 作为更新源） | `.github/workflows/` |
| `fix-backend-open-items` | 服务端更新检查（带缓存、超时、可关闭），状态通过接口暴露，**不让浏览器直连 api.github.com** | Go（service/ common/ setting/ router/ controller/） |
| `review-frontend` | 面板展示"当前版本 / 最新版本 / 是否有更新"（比较逻辑已修好）+ 升级指引 | `web/**` |

## Acceptance Criteria

- [ ] komari 机制的准确描述（文件:行 级证据）+ 可移植性结论（能用/不能用/改哪里）
- [ ] 版本源修好：发版后更新检查不再报错的版本
- [ ] 后端：无浏览器外呼、可关闭、有测试、三库/relaykit 不受影响
- [ ] 前端：当前/最新/有无更新三态正确，unknown 不谎报
- [ ] 全员本地验证命令全绿；Lead 统一复核与提交（等用户发话）
