# Design — deepseek-v4.1-flash 定价接入 + v0.0.44 发布与回滚演练

## 1. 变更边界（写作用域，互不重叠）

| 文件 | 变更 |
|---|---|
| `setting/ratio_setting/model_ratio.go` | `defaultModelRatio`、`defaultCompletionRatio` 各增 `deepseek-v4.1-flash`；`FormatMatchingModelName` 改用共享谓词 |
| `setting/ratio_setting/cache_ratio.go` | `defaultCacheRatio` 增 `deepseek-v4.1-flash` |
| `setting/billing_setting/tiered_billing.go` | `defaultTieredBillingModels` 增 `deepseek-v4.1-flash` |
| `relay/channel/deepseek/constants.go` | `ModelList` 增 `deepseek-v4.1-flash`、`-none`、`-max` |
| `relaykit/relayconvert/reasoning/suffix.go` | 新增导出谓词 `IsDeepSeekV4Model`；`ParseDeepSeekV4ThinkingSuffix` 改用它 |
| `setting/reasoning/suffix.go` | 宿主垫片增一行 re-export（该包契约：relaykit 抽取后宿主 import 路径不变） |
| `relay/common/deepseek_v4_thinking.go` | 两处 `HasPrefix(..., "deepseek-v4-")` 改用谓词 |
| `relay/claude_handler.go` | 一处 `HasPrefix(request.Model, "deepseek-v4-")` 改用谓词 |
| `setting/ratio_setting/model_defaults_test.go` | 家族列表加新模型、目录锁 4→5、新增逐项相等断言 |
| `setting/billing_setting/tiered_billing_defaults_test.go` | 家族列表加新模型 |
| `relay/common/deepseek_v4_thinking_test.go` | 增 `deepseek-v4.1-flash-max` 归一用例（回归锚点） |
| `VERSION` | `0.0.43` → `0.0.44` |

不改：`pkg/billingexpr/**`、峰谷表达式字面量、前端 `web/**`、其余任何模型的价格键。

## 2. 核心设计决定

### 2.1 V4 家族判定收敛为单一谓词

现状：`deepseek-v4-` 前缀字面量散落在 4 个文件 5 处
（`relaykit/.../suffix.go:40`、`relay/common/deepseek_v4_thinking.go:103,119`、
`relay/claude_handler.go:130`、`setting/ratio_setting/model_ratio.go:489`）。
新模型名带**次版本号点号**（`deepseek-v4.1-flash`），这些字面量一处都不会命中——这正是必须收敛的理由。

放在 `relaykit/relayconvert/reasoning`（后缀解析的归属包）导出：

```go
// IsDeepSeekV4Model reports whether modelName belongs to the DeepSeek V4 family
// (deepseek-v4-flash / deepseek-v4-pro / deepseek-v4.1-flash ...): the models that
// accept the -none/-max thinking suffixes and think by default.
func IsDeepSeekV4Model(modelName string) bool {
	if strings.HasPrefix(modelName, "deepseek-v4-") {
		return true
	}
	rest, ok := strings.CutPrefix(modelName, "deepseek-v4.")
	return ok && rest != "" && rest[0] >= '0' && rest[0] <= '9'
}
```

- 依赖方向合法：`relaykit` 不反向依赖根模块（AGENTS.md 硬规则），根模块依赖 `relaykit` 已是现状。
- `deepseek-v4.txt` 这类噪音名不会命中（要求点号后首字符是数字）。
- 5 处调用点改为 `IsDeepSeekV4Model(...)`，字面量只剩谓词内部一处。宿主侧 import 走既有垫片
  `setting/reasoning`（`relay/claude_handler.go`、`setting/ratio_setting/model_ratio.go`），
  `relay/common/deepseek_v4_thinking.go` 保持其既有的 relaykit 直连 import —— 两种既有形态都不新增分叉。

### 2.2 计费语义与家族对齐（显式声明，不新增差异）

| 请求模型名 | 命中的计费路径 | 结果 |
|---|---|---|
| `deepseek-v4.1-flash` | `tiered_expr`（与 `deepseek-flash` 同一条表达式） | 低谷 1/4/0.02，高峰 ×2 |
| `deepseek-v4.1-flash-max` / `-none` | `ratio`（模型/输出倍率经 `FormatMatchingModelName` 归一到基名） | 恒按低谷价 |

第二行是**既有家族行为**：`GetBillingMode` 按请求原名查表（`relay/helper/price.go:79`），后缀变体不在
`defaultTieredBillingModels`，因此走比率价；同理 `GetCacheRatio` 不做归一
（`setting/ratio_setting/cache_ratio.go:47`），后缀变体的缓存倍率回落 1——`deepseek-v4-flash-max` 今天就是这样。
本轮**只对齐、不修正**（修它会改变既有模型行为，违反 R3）；测试里把它锁成"与 `deepseek-v4-flash-max` 一致"，
避免以后被误判为回归。

### 2.3 默认目录规模锁的更新

`TestDefaultCatalogSizeStaysPruned` 断言三张表各 4 键（`model_defaults_test.go:101-104`）。
新增模型后事实是 5 键 → 同步改为 5。这是"刻意改价必须有意为之"的锚点，不能为了过测试而放宽成范围断言。

## 3. 测试设计

| 断言 | 位置 | 保护的不变量 |
|---|---|---|
| 三个数值 + `tiered_expr` + 表达式逐字符相等 | `model_defaults_test.go`、`tiered_billing_defaults_test.go` | 定价口径（R1/AC1/AC2） |
| 五个模型三项数值两两相等（表驱动） | `model_defaults_test.go` | "同价"是可执行断言而非注释 |
| `deepseek-v4.1-flash-max` 归一到基名且价格命中 | `relay/common/deepseek_v4_thinking_test.go` | 后缀接入（R2/AC3） |
| `deepseek-v4-flash-*` 行为不变 | 既有用例保持通过 | 回归面（R3） |
| 目录锁 = 5 | `model_defaults_test.go` | 目录规模 |

不写只提覆盖率的用例；沿用 `require`（setup/致命）+ `assert`（非致命）的既有风格。

## 4. 发布设计（v0.0.44）

1. `VERSION` = `0.0.44`（不带 v），与代码改动同一次提交，锚点 `[task:deepseek-v41-flash-pricing]`。
2. 注释 tag：`git tag -a v0.0.44 -m "v0.0.44"` + 变更要点正文（沿用 v0.0.43 的 changelog 写法）。
3. `git push origin main v0.0.44` → 同时触发 `docker-build.yml`（多架构镜像 + cosign）与
   `release.yml`（Release 条目 + Linux 二进制 + `SHA256SUMS` + `SHA256SUMS.sig`）。
4. 两个工作流并行是既有已知风险（`release.yml` 注释里写明）：Release 成功而镜像失败时，面板会看见
   "有版本没镜像"。**验证必须等到镜像三标签同摘要**，否则按失败处理、重跑 tag 工作流。
5. 一致性核查：`scripts/check-version-drift.sh`（仓库 VERSION / 最新 tag / ghcr 三标签摘要 / 实例版本）。

## 5. 回滚演练设计（本地实例）

| 步 | 动作 | 证据 | 失败恢复 |
|---|---|---|---|
| 0 | 记录升级前二进制 sha256（当前 v0.0.43） | `docker exec new-api sha256sum /new-api` | — |
| 1 | 面板 `apply` 到 v0.0.44 | 响应 `applied/version`；`/api/status` = v0.0.44 | 重跑 apply |
| 2 | 检查回滚点 | `ls -l /new-api.prev`（容器内）+ 状态接口 `rollback_available=true` | 无回滚点则如实记录 |
| 3 | `rollback` 到 v0.0.43 | `rolled_back:true`、版本回退、`sha256(/new-api)` 等于步 0 记录值 | 重跑 apply 回到 v0.0.44 |
| 4 | 再 `apply` 回 v0.0.44 | 版本 = v0.0.44；`.prev` 重建 | 同上 |
| 5 | PAT 调三个 update 端点 | 三个响应均 `pat_not_allowed` | 无系统访问令牌则记为实测缺口 |

- 认证：root 会话，登录接口取 token，**不落盘**（沿用既有会话卫生：本轮结束 logout）。
- 演练本身会把实例短暂回退到 v0.0.43；镜像与 Release 都在，恢复路径明确。
- 该演练行使的是 v0.0.43 已交付能力，本轮不改更新代码。

## 6. 兼容性与回滚

- 代码侧：改动都是"新增键 + 收敛谓词"，revert 单个提交即回到 v0.0.43 行为。
- 数据侧：不涉及迁移、不写数据库；实例侧价格若被运营在 DB 覆盖，代码默认值不影响已存在的 option。
- 实例侧：面板 rollback（本地）或按 README 用固定 tag 重建容器。
