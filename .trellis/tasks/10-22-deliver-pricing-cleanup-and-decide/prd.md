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
