# 排查：模型定价表仍显示 271 个模型

## 现象（2026-10-23）

实例 v0.0.40，管理端「计费与支付 → 模型定价 → 模型价格」表显示 **总计：271**，
列表里仍是 `bge-large-en`、`BLOOMZ-7B`、`chatglm_*`、`chatgpt-4o-latest`、`claude-*` 等旧模型。

## 已知

- 实例 `options`：`ModelRatio` 1 条、`CompletionRatio` 1 条、`CacheRatio` 1 条（已复核）；
- 代码默认：`defaultModelRatio` 10 条、`defaultModelPrice` 20 条（v0.0.40）；
- 对外 `GET /api/pricing` 只返回 1 个模型（abilities 只有 deepseek-flash）。

## 要查（只读，不用浏览器）

1. 该页面的真实数据源：路由（`system-settings/billing/*`）→ 前端 API 定义 → 后端 handler → 数据表/内存 map；
2. 为什么是 271（既不是 240 也不是 10 也不是 1）——列出构成；
3. 给出"让该表只剩 deepseek 家族"的确切操作（代码默认值 / option / 其它缓存）。

## Acceptance Criteria

- [ ] 给出 271 的准确来源（文件:行号 + 数据证据）
- [ ] 给出可执行的收敛方案
- [ ] 实施后该表条目数下降（向用户报告）

## 结论（2026-10-23，只读追查 + 接口实测）

### 271 行的真实来源

「模型定价」表的行 = **8 个 JSON option 键（ModelPrice/ModelRatio/CacheRatio/CreateCacheRatio/CompletionRatio/ImageRatio/AudioRatio/AudioCompletionRatio）
的键名并集**，数据通道：

`model-ratio-visual-editor.tsx:191-220` → `model-pricing-snapshots.ts:146-210`（并集）→ `ratio-settings-card.tsx` →
`section-registry.tsx:100-110 / :25-36`（`getModelDefaults(settings)`）→ `settings-page.tsx:116,123-128` →
`use-system-options.ts:23-29` → `api.ts:34-37`（`GET /api/option/`）→ `controller/option.go:79-116`（**读进程内存 OptionMap**）。

**关键机制**：`/api/option/` 读的是**进程内存**，内存只在**启动**（`model/option.go:143-144`）或**走 API 保存**（`:554-565` → `types/rw_map.go:83-92` **整体替换**）时刷新；
前端对该接口有 **5 分钟 staleTime**（`use-system-options.ts:27`）。

### 实证排除的其它来源

- 前端内置目录**不存在**：下载实例 100 个 chunk / 49 MB，`bge-large-en`、`BLOOMZ-7B`、`models.dev` **0 命中**；
- 代码默认不足以凑出 271：`defaultModelRatio` 9 条、`defaultModelPrice` 20 条；
- `/api/ratio_config` 的 403 是 `ExposeRatioEnabled=false`（`controller/ratio_config.go:12`），与本表无关。

### 接口实测（诊断命令修正后）

上一轮我误把 `/api/option/` 的 `data`（**数组**）当对象用 `to_entries`，导致漏判"没有 ModelRatio" ✗。修正后实测内存值：

| 键 | 条数 |
|---|---|
| ModelPrice | 20（MJ×17 + swap_face + suno_music + suno_lyrics，功能动作计费键）|
| AudioCompletionRatio | 4（tts-1 系列）|
| ModelRatio / CacheRatio / CompletionRatio | 各 1（deepseek-flash）|
| CreateCacheRatio / ImageRatio / AudioRatio | 0 |

**并集 = 25**，与后端自报的 `CompletionRatioMeta` = 25 一致 ✓ → 裁剪已生效，用户截图的 271 是**改配置前打开的旧页面**（5 分钟 staleTime）。

### 用户裁定

功能计费键（20 条）**保留** ✓（删了 Midjourney/Suno 无法预扣费）。

### 操作建议（已给用户）

对「模型定价」页做**强制刷新**（Ctrl+Shift+R），总计应从 271 变为 25。
