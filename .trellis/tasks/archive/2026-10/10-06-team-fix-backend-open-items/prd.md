# 处理后端开放项（团队）：两处历史遗留不一致的评估与处置

## Goal

评估并处理两处历史遗留不一致：能安全改则改并补测试，否则带证据留痕：

1. **multipart 图片编辑缺 `model` 必填校验**：`relay/helper/valid_request.go:195-231` 的 multipart 分支不校验 `model`，
   而 JSON 分支 `:240-243` 有校验。
2. **客户端请求 DTO 非指针标量 + `omitempty`**：`relay/common/relay_info.go:849`（`Duration`）、`dto/video.go:10-12`，
   与 AGENTS.md「可选标量必须用指针 + omitempty」的约定不一致。

## Requirements

- 第 1 项给出**二选一结论**：(a) 补校验（含边界测试：缺 model 400、带 model 正常，并说明为何不破坏合法调用）；
  或 (b) 不改（用代码路径与协议证据说明当前行为可接受）。禁止拍脑袋——每条结论都要能指到代码行。
- 第 2 项先做**影响面清点**（所有读写点与 adaptor，grep 计数），再评估实际后果（显式 0/false 是否真被丢弃、
  是否有默认值兜底）。若波及面可控（读写点 <=10 处且有测试兜底）→ 改成指针 + 补回归测试；
  否则 → 留痕写明后续方案与风险。
- 若涉及时长/计费，必须复核 `relay/relay_task.go:121-127`、`relay/common/relay_utils.go:153` 的钳制仍然成立。

## Acceptance Criteria

- [ ] 两项均有结论（已改 / 不改），每条结论能指到具体 `文件:行` 证据。
- [ ] 若改动：新增/修改测试用 testify（`require` 做致命断言），覆盖边界（缺 model -> 400；带 model -> 正常；
      显式 0/false 不被丢弃）。
- [ ] 四条命令实测并记录退出码：`GOWORK=off go vet ./...`、`GOWORK=off go build ./...`、
      `cd relaykit && GOWORK=off go build ./...`、`make test`。
- [ ] 结论与证据写入本任务目录；向 Lead 汇报（中文，<=30 行，结论先行）。

## Notes

- 禁止网络动作；禁止访问生产；禁止实例化真实 DB dialector（跨库行为用 DryRun + 假 ConnPool 验证）。
- 禁止 `git commit/push/tag`、`git add -A`、git worktree。
- 只改 `relay/ dto/ model/ service/ common/`；不动 `web/**` 与非任务类 `*.md`；不动 `.local-instance/`；不重启服务。
- `relaykit/` 必须独立可构建；JSON 一律走 `common.Marshal/Unmarshal`。
