# 评估：把 x-opencode-session 做进源码（免去每次新建渠道手填）

## 用户问题（2026-10-06）

「只不过，我们每次添加是不是都要这样啊，能不能直接在源码端填上去，评估一下」

## 关键事实（只读调查）

1. 渠道类型 **60 = `ChannelTypeNewAPI`**（`constant/channel.go:60,186`）—— **通用**"New API 兼容"类型，
   **不是** opencode 专用 → 任何按类型硬编码的做法都会**污染所有同类渠道**。
2. **面板已内置 opencode 模板**：`param-override-editor-dialog.tsx:363,379`
   内置 `x-opencode-session` 预设（描述：给不带 session 头的客户端补上）→ 「填入模板」一键插入，**无需手打**。
3. **复制渠道**接口存在：`POST /api/channel/copy/:id`（`router/channel-router.go:73` → `controller.CopyChannel`），
   实现为浅拷贝原渠道（含 `header_override`、含密钥）→ 加同类渠道 = 复制 + 改密钥。
4. 渠道级字段与请求期注入链路完整：`model/channel.go:51`（`HeaderOverride *string`）→ `GetHeaderOverride()`
   → `relay/channel/api_request.go:366/398/422`（正式请求与渠道测试均走此路径；带默认值的占位符在测试中也解析）。
5. 无全局请求头覆盖设置（仅渠道级）；批量编辑不覆盖 `header_override`。
6. 创建渠道的钩子点是 `controller/channel.go:612 AddChannel`。

## 方案评估

| 方案 | 代价 | 结论 |
|---|---|---|
| A. 按类型 60 硬编码 | 极小 | ✗ 否决：类型通用，污染面大；把厂商焊进网关 |
| B. 新增 opencode 专用类型 + adaptor | 中等 | 隔离干净但厂商耦合，上游改要求还得改码 |
| C. 通用「新建渠道默认 header_override」设置（env + 系统设置） | 小（AddChannel 一处 + 设置注册 + 面板一项 + 测试） | ✓ 推荐（若需反复批量新建） |
| D. 用现成能力（模板按钮 + 复制渠道） | 0 | ✓ 立刻可用（零星新建够用） |

## 建议

零星新建 → D；反复批量新建 → C（零厂商耦合、老渠道不受影响）。不做 A；B 仅在确实需要独立类型时考虑。

## 用户决定

（待定）

## 用户决定（2026-10-06）

**先不动代码**。日常新建同类渠道使用面板现成能力：
「填入模板」（内置 `x-opencode-session` 预设，一键插入）+「复制渠道」（`POST /api/channel/copy/:id`，克隆后改密钥）。

若日后需要反复批量新建，再回头做方案 C（通用「新建渠道默认请求头覆盖」）；方案 A（按类型硬编码）已否决。
