# 修 release.yml 的 `isLatest` 不可用字段

## Goal

`v0.0.7` 的 Release **实际已成功发布**（`gh release list` 显示 v0.0.7 = Latest，三个资产齐备），
但 run 被判红——失败在**最后一步的校验字段**：

```
Unknown JSON field: "isLatest"
Available fields: … isDraft, isImmutable, isPrerelease, name, publishedAt, tagName …
```

即 `gh release view --json isLatest` 在该 `gh` 版本上不存在。修掉它，并审计同类问题。

## Requirements

### R1 用**存在的字段或等价判定**替换 `isLatest`

优先采用与面板**同一个端点**的判定（面板读的就是 `releases/latest`）：

```bash
gh api "repos/${GITHUB_REPOSITORY}/releases/latest" --jq .tag_name
```

再与目标 `$TAG` 比对。这样断言的是**面板真正会看到的东西**，而不是 `gh` 的本地字段。
（`gh release list --limit 1` 是按 `created_at` 排序，**不等于** latest flag，因此不采用。）

### R2 审计同段脚本内其它 `--json` / `--jq` 字段

全文件普查，确认没有别的不可用字段被这一步掩盖。

### R3 三种情况都必须可证明

| 情况 | 期望 |
|---|---|
| 目标 tag **就是** latest | ✅ 通过 |
| 目标 tag **不是** latest（重跑旧 tag，`--latest=false`） | ✅ 通过且不误报 |
| 查询**失败** / 返回空 | ❌ **报错退出，绝不静默通过** |

补充：**prerelease 必须不成为 latest**——若 prerelease 出现在 `releases/latest`，应报错。

### R4 不动已发布的 v0.0.7

不再触发工作流、不重推 tag、不动任何 release 资产。

## 验证方式

- `bash -n` 语法检查全部 `run:` 脚本；
- 构造 `gh` **stub**（伪造 `gh api … --jq .tag_name` 的输出/退出码），对三种情况逐一验证；
- 断言工作流内**不再出现** `isLatest`。

## Constraints（硬约束）

- 写范围**仅** `.github/workflows/release.yml` + 本任务目录；
- 不 commit / 不 push（Lead 统一提交）；
- **不触发工作流、不重推 tag、不动 v0.0.7**；
- 只读外部网络。

## Acceptance Criteria

- [ ] `isLatest` 从工作流中彻底移除，改用 `releases/latest` 等价判定
- [ ] 非 latest 的旧 tag 重跑不会被误判为失败
- [ ] 查询失败时 `::error::` + 非零退出（有实测输出）
- [ ] prerelease 出现在 `releases/latest` 时报错
- [ ] 全文件其它 `--json`/`--jq` 字段已审计并列出
- [ ] `bash -n` 全 PASS
- [ ] 未触发工作流、未动 v0.0.7
