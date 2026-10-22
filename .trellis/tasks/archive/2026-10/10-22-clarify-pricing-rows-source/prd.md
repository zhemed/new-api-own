# 澄清模型定价表剩余 25 行的来源

## 背景

用户强刷后表格显示 **总计 25**（截图：末几行是 `tts-1`/`tts-1-1106`/`tts-1-hd`/`tts-1-hd-1106`，价格列显示「未设/未设」），
并问：「这还有你测试剩下的残留吧」；同时上一条指示是「保留那 20 个功能计费键也移除吧」。

## 要查（只读）

1. 数据库 `options` 里现存的定价键（= 我写过的）与**不存在**的键（= 由代码默认值在进程内存里播种）分别有哪些；
2. 代码默认表 v0.0.40 的构成：`defaultModelRatio` / `defaultModelPrice` / `defaultAudioCompletionRatio` 等各自的条目；
3. 结论：25 行里哪些来自**我的写入**、哪些来自**产品默认**；`tts-1*` 为何显示"未设价格"。

## 待用户确认后执行

- 是否清空 20 个功能计费键（`ModelPrice` → `{}`，代价：Midjourney/Suno 无法按次预扣费）；
- 是否同时清空 4 个音频键（`AudioCompletionRatio` → `{}`，代价：未带 model 的语音/转写请求会报未配置计费）。

## Acceptance Criteria

- [ ] 每行来源有据（数据库 vs 代码默认，带证据）
- [ ] 用户确认后执行并复核表格总计

## 核实结论（2026-10-23）

### 25 行的来源拆解

| 行 | 来源 | 证据 |
|---|---|---|
| `deepseek-flash` | **实例写入**（DB `ModelRatio`）| DB 有该键，1 条 |
| 20 行 `mj_*`(17)/`swap_face`/`suno_music`/`suno_lyrics` | **产品默认值** | DB **无** `ModelPrice` 键；`defaultModelPrice` 20 条 |
| 4 行 `tts-1*` | **产品默认值** | DB **无** `AudioRatio`/`AudioCompletionRatio`；`defaultAudioRatio`/`defaultAudioCompletionRatio` 各 4 条 |

⇒ **不是测试残留**；我实际写入库的只有 3 个键（`ModelRatio`/`CompletionRatio`/`CacheRatio`，各 1 条）。

### 我的副作用（如实认领）

把 DB `ModelRatio` 覆盖为 1 条后，代码默认的 9 条（含音频 4 条与 `whisper-1`）被**整体替换** →
音频模型**丢失价格**（表格显示"未设"）→ 真调用语音接口会报未配置计费。

### 用户裁定与执行

用户选择「两批都清」→ 已写入 3 个 option 并复核：

| 键 | 现值 |
|---|---|
| `ModelPrice` | `{}`（清空 20 个 MJ/Suno/换脸键）|
| `AudioRatio` / `AudioCompletionRatio` | `{}`（清空 4 个音频键）|
| `ModelRatio` / `CompletionRatio` / `CacheRatio` | 各 1 条（deepseek-flash）|

**接口内存并集 = 1**，唯一模型名 = `deepseek-flash` ✓（`CompletionRatioMeta` 亦为 1）。

### 代价（用户已知情并接受）

Midjourney/Suno 无法按次预扣费；未带 model 的语音/转写请求会报未配置计费。

### 遗留

代码默认表仍含 `defaultModelPrice` 20 条与音频 4 条 → **新部署**或点"重置比率"会重新出现（本实例不受影响）。
