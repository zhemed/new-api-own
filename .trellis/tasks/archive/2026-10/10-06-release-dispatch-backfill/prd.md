# release.yml 支持手动补发 + 记录并行风险

## Goal

1. 给 `.github/workflows/release.yml` 加 `workflow_dispatch`（必填 `tag` 输入），使**存量 `v0.0.6`
   无需重推 / 改写已发布 tag** 就能补出 Release。
2. 把"与 `docker-build.yml` 同 tag 并行"的风险写进工作流注释。

## Requirements

### R1 `workflow_dispatch`（补发通道）

- 新增 `workflow_dispatch.inputs.tag`：**必填**、`type: string`。
- **push 路径与手动路径复用同一 job、同一段脚本**，不得复制两份发布逻辑。
- 手动路径同样遵守既有全部规则：
  - 只允许版本 tag（`v[0-9]*` 或 `[0-9]*`），其它一律**拒绝并清晰报错**；
  - 带 `-` 的 tag 走 `--prerelease` 且**永不** `--latest`；
  - 仅"最高版本 tag"可拿 `--latest`，其余 `--latest=false`；
  - `gh release view` → `edit` 幂等，不重复建 Release；
  - `--verify-tag` 保证 tag 真实存在。
- **脚本注入防护**：`github.event.inputs.tag` 必须经 `env:` 传递，禁止直接内插进 `run:` 脚本。

### R2 并行风险注释

在文件头注释写明：`release.yml` 与 `docker-build.yml` 同 tag 并行触发，若镜像推送失败而 Release
成功，会对外提示一个拉不到的版本；**彻底闭环需 `workflow_run` 链式**，
**本轮有意保留 `on: push` 以降低耦合**。

### R3 校验

- YAML 解析通过。
- 贴出**非法 tag（如 `release-notes`）被拒绝**的实际输出。
- 贴出合法/非法 tag 的判定矩阵。
- `docker-build.yml` 未被改动（本任务只动 `release.yml`）。

## Constraints（硬约束）

- 写范围**仅** `.github/workflows/` + 本任务目录；
- **不 push、不触发任何工作流**；
- 不 commit；只读外部网络；
- 不碰生产机与本机 3020、不碰 `.local-instance/`。

## 暂不做（等用户定）

- 清理旧 `v0.0.5` Release 上的 118MB 二进制资产；
- 存量 `v0.0.6` 的**实际**补发（前置条件：本工作流先落到默认分支 + 用户同意提交推送）。

## Acceptance Criteria

- [ ] `release.yml` 有 `workflow_dispatch` + 必填 `tag` 输入
- [ ] push 与手动路径共用同一 job、同一段发布脚本（无重复逻辑）
- [ ] 非法 tag 被拒绝且有清晰报错（有实测输出）
- [ ] 手动路径的 prerelease / latest / 幂等 / verify-tag 规则与 push 路径一致
- [ ] 并行风险已写入注释，含"需 `workflow_run` 链式、本轮有意保留 `on: push`"
- [ ] YAML 解析通过；`docker-build.yml` 未改；未 push、未触发工作流

## Notes

- 来源：Lead 指派（承接 `10-06-komari-update-and-version-source`）。
- 关键背景：Lead 明确**不接受改写已发布的 tag**，故必须有手动补发通道。
