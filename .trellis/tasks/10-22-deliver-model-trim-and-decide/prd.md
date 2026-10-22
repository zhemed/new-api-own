# 交付模型目录裁剪与官方定价

1. 汇报实例侧（option 裁剪 + 峰谷表达式）与代码侧（v0.0.40）成果；
2. 询问：是否升级实例、是否再删 5 个音频默认与 3 个别名、渠道上游失败是否要我排查。

## 用户裁定与执行结果（2026-10-23）

| 问题 | 用户裁定 | 执行 |
|---|---|---|
| 实例升级 | 升到 v0.0.40 | 已升级（RestartCount 0）；重启后 option 未被覆盖：ModelRatio/CompletionRatio/CacheRatio 各 1 条、billing_mode 为 tiered_expr、币种仍 CUSTOM+¥ |
| 目录里保留的 8 个非 deepseek 键 | 都保留 | 保持（5 个音频默认 + 3 个 deepseek 别名，均有硬依赖依据）|
| 渠道上游报错 | 「不是你的原因，是我自己忘记更新 api，可以归档了」 | 归档，不再排查 |

## 最终状态

- 实例：v0.0.40；`ModelRatio` 240 → **1** 条、`CompletionRatio` 7 → 1、`CacheRatio` 76 → 1、峰谷表达式计费已启用；
- 代码：默认目录 239 → 10（比率）、31 → 20（固定价，均为功能动作键），deepseek-flash 默认价为官方低谷价（0.5/4/0.02）；
- 备份：`/root/data/one-api.db.bak-before-model-trim-*` 保留（可回滚）；
- 临时令牌：全部清理，仅剩用户自己的 1 个。
