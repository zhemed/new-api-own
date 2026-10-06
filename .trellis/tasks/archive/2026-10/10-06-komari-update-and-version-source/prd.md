# 读 komari 更新逻辑并修好版本源

## Goal

1. 准确描述 `https://github.com/zhemed/komari` 的**在线更新机制**（文件:行 级证据），
   判断其中哪些做法可移植到本项目（`docker run` 单容器 + 公开镜像）。
2. **修好版本源**：生产机"检查更新"报 **"有新版本可用：v0.0.5"**，实际最新是 `0.0.6`。
   根因：面板读 **GitHub Releases**，而我们的 **Releases 停在 `v0.0.5`**
   —— `v0.0.6` 只有镜像、没有 Release 条目（Release 工作流在瘦身时被删）。

## Requirements

### R1 komari 更新机制调研（只读）

必须给出**文件:行 级**证据，覆盖：

- **更新源**：GitHub Releases / GitHub API / 自有服务？端点、鉴权、缓存、限流处理；
- **版本比较**：semver？`v` 前缀处理？预发布（-alpha/-beta/-rc）如何排序？失败如何降级；
- **是否真正自更新**：下载/替换二进制/重启/容器重建，还是只做"检查 + 提示 + 指引"；
- **可移植性结论**：我们这套 `docker run` 单容器部署**能用哪些、不能用哪些**。
  **必须正面回答**：在容器内替换二进制会被 recreate 丢掉。

抓取方式：`web_fetch` 仓库页 / `raw.githubusercontent.com` / `api.github.com`（均已授权，只读）。
**不 clone**、不跑外部工具。

### R2 修好版本源（写范围：`.github/workflows/`）

方案自选并给理由：

- **(a) 恢复"仅镜像"Release 工作流**：tag 触发，写发布说明 + 指向镜像，**不放二进制**，
  与"唯一部署方式 = 镜像"一致；
- **(b) 让更新检查改读 registry**（ghcr 标签/摘要）：若能实现则与后端成员范围重叠
  → **只给结论、不动 Go**，交由 Lead 转后端成员。

若选 (a)：写 `.github/workflows/release.yml`：
- `on: push: tags: v[0-9]*`（与已修的 `docker-build.yml` 触发条件一致）
- 权限最小化（`contents: write`）
- `softprops/action-gh-release` 或 `gh release create` 二选一
- **不与 `docker-build.yml` 重复建 Release**
- YAML 解析校验通过

## Constraints（硬约束）

- 只读外部：**仅** `github.com/zhemed/komari`、`api.github.com`、`ghcr.io`（只读）；
  **其它外部主机一律不碰**；
- 不 commit / 不 push / 不触发任何工作流；
- 不碰生产机、不碰本机 3020、不碰 `.local-instance/`；
- 改文件**只限 `.github/workflows/`**；
- 不 `git add -A`、不建 worktree。

## Acceptance Criteria

- [ ] komari 机制表含文件:行级证据，四项（源/比较/自更新/降级）齐全
- [ ] 明确回答"容器内替换二进制会被 recreate 丢掉"这一条
- [ ] 版本源方案(a)/(b) 有选择理由，且与"唯一部署方式 = 镜像"不冲突
- [ ] 若落地 (a)：`release.yml` 存在、YAML 解析通过、触发条件与 `docker-build.yml` 一致、
      不重复建 Release、不放二进制
- [ ] 未 push、未触发工作流、未碰生产与本机 3020

## Notes

- 用户点名任务，Lead `lead` 指派。
- 写范围仅 `.github/workflows/` + 本任务目录。
