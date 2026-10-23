# Implement — deepseek-v4.1-flash 定价接入 + v0.0.44 发布与回滚演练

## 阶段与依赖

```
P1 代码与单测 ──► G1 独立验证 ──► P2 提交与发版 ──► G2 产物校验 ──► P3 本地演练 ──► G4 收尾留痕
   (worker)        (verifier)       (Lead 执行)      (verifier)       (worker)        (Lead)
```

- P1 与 P3 的写作用域不重叠：P1 只改 Go 源码与测试，P3 只碰运行中的实例与容器。
- P2 的提交与推送是**对外动作**，由 Lead 执行（本轮已获用户授权"改完发版"）。
- 每个 G 门未过不得进入下一阶段；打回时用 `send_message` 唤醒同一成员续改。

## P1 代码与单测（写作用域见 design.md 第 1 节）

1. `relaykit/relayconvert/reasoning/suffix.go`：新增导出谓词 `IsDeepSeekV4Model`（实现见 design 2.1），
   `ParseDeepSeekV4ThinkingSuffix` 的前缀判定改用它。
2. `relay/common/deepseek_v4_thinking.go:103,119` 与 `relay/claude_handler.go:130`：改用该谓词。
3. `setting/ratio_setting/model_ratio.go`：`FormatMatchingModelName` 的前缀判定改用该谓词；
   `defaultModelRatio` 增 `"deepseek-v4.1-flash": 0.5`、`defaultCompletionRatio` 增 `...: 4`。
4. `setting/ratio_setting/cache_ratio.go`：`defaultCacheRatio` 增 `"deepseek-v4.1-flash": 0.02`。
5. `setting/billing_setting/tiered_billing.go`：`defaultTieredBillingModels` 增 `deepseek-v4.1-flash`。
6. `relay/channel/deepseek/constants.go`：`ModelList` 增 `deepseek-v4.1-flash`、`deepseek-v4.1-flash-none`、
   `deepseek-v4.1-flash-max`（与 `deepseek-v4-flash` 三件套对齐）。
7. 测试：
   - `setting/ratio_setting/model_defaults_test.go`：家族列表两处各加新模型；目录锁 4→5；
     新增"五个模型三项数值两两相等 + `tiered_expr` + 表达式逐字符相等"的表驱动断言。
   - `setting/billing_setting/tiered_billing_defaults_test.go`：家族列表加新模型。
   - `relay/common/deepseek_v4_thinking_test.go`：增 `deepseek-v4.1-flash-max` → 基名 + `enabled/max`
     的归一用例；并断言与 `deepseek-v4-flash-max` 结果一致。

**验证命令（P1 出口，逐条必须 exit 0）**

```bash
GOWORK=off go vet ./...
GOWORK=off go build ./...
make test
cd relaykit && GOWORK=off go build ./...
gofmt -l setting relay relaykit/relayconvert      # 必须无输出
```

**G1 独立验证（verifier，不写代码）**

- 自己重跑上述命令，不看 worker 的转述；
- 交叉检查：三张表 + tiered 列表 + ModelList 的键是否齐全；谓词改动的 5 处调用点是否都改了、
  有无遗留 `"deepseek-v4-"` 字面量（`grep -rn '"deepseek-v4-' --include=*.go .` 只应命中谓词内部）；
- 反例检查：`deepseek-v4.1-flash`（无后缀）不进 `ratio` 路径；`deepseek-v4.1-flash-max` 的
  `GetBillingMode` 是 `ratio` 且与 `deepseek-v4-flash-max` 一致（既有家族行为，见 design 2.2）；
- 回归检查：`deepseek-v4-pro`、`deepseek-v4-flash-*`、非 deepseek 模型（如 `gpt-4o`）行为不变。

## P2 提交与发版

8. `VERSION`：`0.0.43` → `0.0.44`（不带 v）。
9. 提交（锚点必需）：`git add -A && git commit -m "feat(pricing): 新增 deepseek-v4.1-flash（与 deepseek-flash 同价）[task:deepseek-v41-flash-pricing]"`。
10. 注释 tag + 推送：`git tag -a v0.0.44 -m "v0.0.44" -m "<changelog 要点>"`；`git push origin main v0.0.44`。
11. 等 CI 两个工作流（`docker-build.yml`、`release.yml`）跑完；失败则看日志判断是重跑还是补提交。

**G2 产物校验（verifier）**

```bash
make check-version INSTANCE=http://127.0.0.1:3000
docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:latest \
  | grep -i digest            # 与 :v0.0.44 / :0.0.44 三处摘要一致
gh release view v0.0.44 --repo zhemed/new-api-own --json assets   # 含 SHA256SUMS / SHA256SUMS.sig / 两个二进制
```

- 三标签摘要不一致或 Release 缺资产 → 不通过（`release.yml` 与镜像并行是既有风险，见 design 4.4）。

## P3 本地实例演练（worker；认证凭据由 Lead 向用户取，不入库、不落盘）

12. 步 0：`docker exec new-api sha256sum /new-api`（记录 v0.0.43 基线哈希）。
13. 步 1：root 会话 → `POST /api/status/update/apply` → `/api/status` 版本 = v0.0.44。
14. 步 2：`docker exec new-api ls -l /new-api.prev` + `GET /api/status/update` → `rollback_available:true`。
15. 步 3：`POST /api/status/update/rollback` → `rolled_back:true`、版本回 v0.0.43、
    `/new-api` 的 sha256 == 步 0 记录值。
16. 步 4：再 `apply` → 版本回 v0.0.44，`.prev` 重建。
17. 步 5：用系统访问令牌调三个 update 端点（GET status、POST apply、POST rollback）→ 均 `pat_not_allowed`。
18. 会话卫生：演练结束 `POST /api/user/auth/logout`。

**G3 演练证据**：五步的原始输出（版本号、布尔值、sha256、错误码）逐条贴回 Lead 复核；
任何一步失败不得改代码"绕过"，按 design 第 5 节的恢复路径处理并如实记录。

## P4 收尾

19. `python3 .trellis/scripts/task.py finish` → `task.py archive 10-23-deepseek-v41-flash-pricing --skip-branch-validation`。
20. 写 journal（`.trellis/workspace/new-api-own/journal-2.md`）+ 提交留痕。
21. 若发现值得沉淀的约定（如"家族前缀判定必须走谓词"），按 `trellis-update-spec` 写入 spec。

## 回滚点

| 阶段 | 回滚手段 |
|---|---|
| P1 未提交 | `git checkout -- <files>` |
| P2 已提交未推 tag | `git reset`/新提交；tag 未推可删 |
| P2 tag 已推 | 不改历史：补发 v0.0.45（README/MAINTENANCE 既有做法） |
| P3 实例 | 面板 `apply` 回 v0.0.44；必要时按 README 用固定 tag 重建容器 |

## 验收对照

| 计划步骤 | 覆盖的 AC |
|---|---|
| 步骤 3-5 | AC1、AC2、AC6 |
| 步骤 1-2、7 | AC3、AC4 |
| 步骤 出口命令 | AC5 |
| 步骤 8-11、G2 | AC7 |
| 步骤 12-18 | AC8 |
| 全流程 | AC9 |
