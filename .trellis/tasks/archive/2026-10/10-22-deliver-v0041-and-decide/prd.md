# 交付 v0.0.41

代码默认表只剩 deepseek 家族（4/0/4/4）；实例表格已只剩 deepseek-flash。
待用户决定：升级实例；是否把 distributor 的默认音频模型注入改为直接报错。

## 执行结果（2026-10-23）

| 项 | 用户裁定 | 执行 |
|---|---|---|
| 实例升级 | 升到 v0.0.41 | 已升级（RestartCount 0），币种与定价 option 重启后保持 |
| 音频注入语义 | 保持现状 | 不改（未带 model 的音频请求仍是 503 无渠道 / 400 未配置计费，语义明确）|

### 升级后复核（数据库 + 接口双证）

| 键 | 值 |
|---|---|
| ModelRatio | 1 条（deepseek-flash）|
| CompletionRatio / CacheRatio | 各 1 条（deepseek-flash）|
| ModelPrice / AudioRatio / AudioCompletionRatio | **0 条**（空）|

- 实例库里出现的模型名**并集 = `deepseek-flash`** ✓；
- 接口 8 键**并集 = 1**，唯一名称 `deepseek-flash` ✓；
- 币种：`CUSTOM` + `¥` ✓。
