# Implement — deepseek 家族判定与默认模板（v0.0.45）

## 阶段

```
P1 代码与测试 ──► G1 独立验证 ──► P2 提交发版 ──► G2 产物校验 ──► P4 留痕收尾
   (worker)         (verifier)       (Lead)          (verifier)        (Lead)
```

## P1 代码与单测（写作用域见 design.md 第 1 节）

1. `relaykit/relayconvert/reasoning/suffix.go`：`IsDeepSeekV4Model` → `IsDeepSeekModel`，
   判定简化为 `strings.HasPrefix(modelName, "deepseek")`，注释说明"家族前缀 + 默认思考"两条语义；
   `ParseDeepSeekV4ThinkingSuffix` 改调用新名。
2. `setting/reasoning/suffix.go`：re-export 行改名。
3. `relay/common/deepseek_v4_thinking.go`：2 处调用点改名；三个导出函数注释改为 deepseek 家族口径。
4. `relay/claude_handler.go`、`setting/ratio_setting/model_ratio.go`：调用点改名。
5. 默认模板裁剪（四张表口径必须一致）：
   - `model_ratio.go`：`defaultModelRatio`、`defaultCompletionRatio` 各留 2 键；
   - `cache_ratio.go`：`defaultCacheRatio` 留 2 键；
   - `tiered_billing.go`：`defaultTieredBillingModels` 留 2 个模型；
   - 相关注释里的别名列举同步。
6. `relay/channel/deepseek/constants.go`：`ModelList` = 6 条（见 prd R3）。
7. 测试：
   - `model_defaults_test.go`：目录锁 2；家族列表两个模型；同价断言只对这两个模型；
     被裁掉的 `deepseek-v4-flash` / `deepseek-chat` / `deepseek-reasoner` 补进"不应再有默认价"的用例。
   - `tiered_billing_defaults_test.go`：家族列表两个模型。
   - `relaykit/.../suffix_test.go`：真值表按新语义重写；`deepseek-v4.txt` 一类旧边界断言删除。
   - `relay/common/deepseek_v4_thinking_test.go`：新增 `deepseek-flash` / `-max` / `-none` 三档用例，
     并与 `deepseek-v4-flash` 同后缀结果逐项对照。
8. `MAINTENANCE.md` 默认目录段落同步为两个模型；`VERSION` → 0.0.45。

**P1 出口命令（逐条 exit 0，附原始输出）**

```bash
GOWORK=off go vet ./...
GOWORK=off go build ./...
GOWORK=off go test -count=1 ./setting/... ./relay/...
cd relaykit && GOWORK=off go test -count=1 ./... && GOWORK=off go build ./...
gofmt -l <改动的每个文件>            # 无输出
```

（全量 `make test` 由 Lead 在 G2 前用 `-count=1` 复跑；`make test` 默认吃缓存，不作为证据。）

## G1 独立验证（verifier）

- 自己复跑出口命令；核对 `git status` 只含声明范围内的文件；
- 反例：`gpt-4o`、`DeepSeek-flash`（大写）、空串、`deepseek-flash-high`（不支持的档位）；
- 正例：`deepseek-flash`、`deepseek-flash-max`、`deepseek-flash-none`、`deepseek-v4.1-flash-max`
  在 OpenAI/Claude/Responses 三条路径上的结果，且与 `deepseek-v4-flash` 同后缀一致；
- 一致性：四张默认表的键集完全相同 = 2；`ModelList` 恰好 6 条且与两张基名一致；
- mutation：把谓词改成 `false`（应让正例全红）、把某张表多留一个键（应让目录锁红），改完还原。

## P2 提交与发版（Lead）

9. `git add -A`（含任务目录）→ commit 锚点 `[task:deepseek-family-match-and-trim]`；
10. `git tag -a v0.0.45 -m ...` → `git push origin main v0.0.45`；
11. 等 `docker-build.yml` + `release.yml` 完成。

**G2 产物校验**

```bash
for t in latest v0.0.45 0.0.45; do docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:$t | grep -i ^Digest; done
gh release view v0.0.45 --repo zhemed/new-api-own --json assets
./scripts/check-version-drift.sh          # 不带 --instance（该脚本实例比对有既有 v 前缀缺陷）
```

## P4 留痕

12. 写本任务 prd 的验收结果；journal；`task.py finish` → `archive --skip-branch-validation`。

## 回滚点

| 阶段 | 手段 |
|---|---|
| P1 未提交 | `git checkout -- <files>` |
| tag 已推 | 不改历史：补发 v0.0.46 |
| 语义要退回 | revert 本次提交即回到 v0.0.44 语义 |
