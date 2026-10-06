# 团队复查本轮改动

## 背景（被复查的一轮，commit 208483c / 97fd7d6 / 5ad97b7）

- 装了 bun 1.4.2 + web 依赖 + jsdom/@types/jsdom；
- 修了我们自己引入的 lint error（`upstream-conflict-dialog` 的 useMemo 缺 `t` 依赖）；
- **改了消毒逻辑**：`footer.tsx` 在 `DOMPurify.isSupported === false` 时返回空串（原来会原样注入）；
- **改了测试基建**：`footer.test.tsx` 由 happy-dom 换 jsdom（断言未改）；
- **改了 knip 配置**：声明应用入口 `index.html`/`src/main.tsx` + 测试入口，`ai-elements` 登记 ignore；
- **删除 33 个前端文件**（两轮 knip + 引用保险过滤 + `bun run build` 兜底）；
- `.gitignore` 增加 `.verify-*/`（清掉误提交的 gitlink）。

## 复查分工（写范围互不重叠）

| 成员 | 共享任务 | 写范围 | 重点 |
|---|---|---|---|
| `audit-frontend-writes` | task-5 | `web/**` | 33 个删除是否真的安全（**字符串式/动态/约定式引用**：`import(` 动态、i18n static-keys、路由约定、registry、CSS/HTML 引用）；knip 配置是否过宽；jsdom 改动是否弱化断言；消毒加固是否引入行为缺陷 |
| `audit-backend-docs` | task-6 | Go 代码 + `.md` + `.trellis/spec/` | Go 侧确未受影响（vet/build/test/relaykit）；全仓库是否还有指向已删文件的残留引用；MAINTENANCE 的工具链段/已知不一致段描述是否准确 |

## 硬约束（全员）

- 禁止网络动作；禁止访问生产；禁止 git commit/push/tag；**禁止 `git add -A`、禁止在仓库内建 git worktree**（上一轮就在这两处翻过车）；不要动 `.local-instance/` 与 3020 端口上的演示实例（**不要重启任何服务**）。
- 结论必须附证据（文件:行）；不确定标"待确认"；只报真实问题，不报风格偏好。

## Acceptance Criteria

- [ ] 两条线各自给出：核验方法、证据、结论（安全/不安全）
- [ ] 发现的问题分"已修（限本范围）"与"需 Lead 处置"
- [ ] 明确说明哪些结论无法验证及原因
