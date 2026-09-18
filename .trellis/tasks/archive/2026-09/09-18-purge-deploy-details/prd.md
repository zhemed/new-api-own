# 清除仓库内的部署细节污染（内网 IP/主机/实例信息）

## 结论

工作树与**所有可达历史**（main、tags、全部提交内容与提交消息）已无部署细节；
旧对象在 GitHub 上仍可按 SHA 访问（force push 不清理悬空对象），彻底清除需联系 GitHub Support。

## 做了什么

### 1. 工作树清理（提交 `d2ca7cf`，原 `635eac6`）

- `MAINTENANCE.md`：删掉内网 IP、主机名、线上实例域名、凭据路径，措辞改为通用描述
- `.trellis/` 归档任务 PRD / `task.json`、`workspace` 下 journal 与 index：内网 IP、线上域名、
  实例专属兜底值、渠道名、运维备份路径全部脱敏
- 新增第六条铁律 +「仓库边界」章节：可写（RFC1918 示例、占位口令、通用路径）与
  不可写（真实内网 IP/域名/主机名、实例专属值、凭据位置、运维机绝对路径）对照表 + 提交前自检命令
- `AGENTS.md` 的 `TRELLIS-GATE` 段同步加入该条，约束后续 AI 会话不得再写入
- 保留项：`common/ssrf_protection.go`、`middleware/trusted_proxies.go`、`.env.example`、
  `docs/authentication.md`、i18n 词条中的 RFC1918 —— 均为通用示例

### 2. 历史重写（`git filter-repo`，两轮 `--replace-text` + `--replace-message`）

- 备份：`/tmp/new-api-own-backup.git`（`git clone --mirror`，含全部 refs/tags，110 个提交）
- 第一轮替换 7 个字符串；第二轮补掉裸串（第一轮只覆盖了带后缀的完整值）
- 验证方式：对每个目标串做 `git rev-list --all` 逐提交 `git grep`，全部 0 处；
  提交消息扫描 0 处；`go.sum` 里的 base64 哈希巧合（含目标字串的随机片段）**未被误改**
- 主分支与两个 tag 强推成功：`main → cc6816e`、`v0.0.2 → 4d95e4f`、`v0.0.3 → 8a99114`
- `.trellis/gates/enforce-from` 更新为重写后的闸门提交 SHA（`3b658688…`），本地与 CI 审计均通过

### 3. 重写后的状态验证

- GitHub Releases **未受影响**：`v0.0.3`（Latest，7 个附件齐全）与 `v0.0.2` 均在
- CI 全部重跑并通过：`Release`、`Build Electron App`、`Publish Docker image (Multi-arch)`、
  `trellis-gate`（新 HEAD `cc6816e`）
- 本地与远端一致（`origin/main...HEAD` = 0/0）

## 局限（诚实写）

- **悬空对象仍在**：旧提交按 SHA 仍可通过 GitHub API 取到内容（已实测 `4a24792` 的
  `docker-compose.yml` 仍能拉到含内网 IP 的版本）。GitHub 的既定行为是 force push 不清理
  不可达对象；要彻底清除需按官方流程联系 GitHub Support 请求 purge（提供仓库名与旧 SHA）。
- 本机仍保留旧历史的副本：`/tmp/new-api-own-backup.git`，以及更早的 `/tmp/fresh-clone`、`/tmp/fresh2`
  测试克隆（均在 `/tmp`，不进仓库；需要彻底清掉可以删除）。
- 已发布的容器镜像内容不受影响（镜像内只含二进制与许可证文件，不含仓库文档）。

## Acceptance Criteria

- [x] 工作树内无部署细节（扫描 0 命中）
- [x] 建立「部署细节不进仓库」规则 + 自检命令，写入 `MAINTENANCE.md` 铁律与 `AGENTS.md`
- [x] 历史重写完成并强推；全部可达提交的内容与消息中目标串 0 命中
- [x] Releases / CI 在重写后完好并通过
- [x] 说明悬空对象与本地备份的残留，给出彻底清除的路径（GitHub Support）
