# 交付定价表清理结果并询问代码默认表

## 已完成（实例侧）

用户强刷后表格显示 25 行；经核实 24 行来自产品默认目录（非测试残留），并按用户裁定清空：

| 键 | 现值 | 说明 |
|---|---|---|
| ModelPrice | `{}` | 清空 20 个 MJ/Suno/换脸功能计费键 |
| AudioRatio / AudioCompletionRatio | `{}` | 清空 4 个 tts-1 音频键 |
| ModelRatio / CompletionRatio / CacheRatio | 各 1 条 | 仅 deepseek-flash |

接口内存 8 键并集 = **1**，唯一模型名 `deepseek-flash`。

## 本轮（用户裁定"代码默认也一并清掉"）

- 代码默认表清空 `defaultModelPrice`(20)、`defaultAudioRatio`(4)、`defaultAudioCompletionRatio`(4)，并从 `defaultModelRatio` 移除 `whisper-1` 与 4 个 tts（**保留** deepseek-flash + 3 别名）；
- 影响面：新部署与"重置比率"后表格同样只剩 deepseek 家族；
- 代价（用户已知情）：MJ/Suno 无法预扣费；未带 model 的语音/转写请求走未配置计费。

## Acceptance Criteria

- [ ] 代码清空后 Go 门禁全绿（vet/build/relaykit 构建/make test）
- [ ] 发版并在实例验证（"重置比率"不会带回旧键）

## 执行结果（2026-10-23，v0.0.41）

用户裁定"代码默认也一并清掉" → 已实现并发布：

| 默认表 | 改动 |
|---|---|
| `defaultModelRatio` | 9 → **4**（`deepseek-flash` + 3 个别名；移除 `whisper-1` 与 4 个 tts）|
| `defaultModelPrice` | 20 → **空 map** |
| `defaultAudioRatio` / `defaultAudioCompletionRatio` | 4 / 4 → **空 map** |
| `defaultCompletionRatio` / `defaultCacheRatio` | 4 / 4（未动）|

- 文件核对（Lead 亲自读 `model_ratio.go:16-60`）：三张表确为空 map，注释写明产品口径 ✓；
- 测试：规模锁定改为 4/0/4/4 + 新增"默认表只含 deepseek 家族"；`make test` **41 包 ok** ✓；

### 注入链路的最终行为（成员只读核实，未改语义）

客户端**未带 model** 的音频请求：先过渠道门（本实例只绑 deepseek-flash）→ **503 无可用渠道**；
若有渠道服务该模型但未配价 → **400「模型价格未配置」**（`relay/helper/price.go:101-110`）；
**不会**静默按 37.5 兜底计费（仅 self-use 或用户显式勾选 AcceptUnsetRatioModel 才走兜底）。

### 副作用（用户已知情）

音频端点与 MJ/Suno 功能需要运营在实例侧自行配价，否则请求被拒（非 self-use）。

### Lead 自报

我的抽查脚本用非贪婪正则跨读了下一个 map，一度误显示"五张表都含 deepseek 4 条"；**改用直接读文件核实**，确认代码正确（教训：结构性断言不要用跨行正则，读文件或用测试）。
