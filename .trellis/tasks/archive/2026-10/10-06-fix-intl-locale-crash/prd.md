# 修复 v0.0.8 回归：日志维护页在 zhCN 语言标签下整页崩溃

## 现象（用户实例上实测发现）

`/system-settings/operations/logs` 整页显示错误边界「500 糟糕！出错了 :')」；
同一批其它运营子页（behavior/alerts/email/worker/performance/update-checker）全部正常 —— 只有我们改过的那个组件崩。

## 根因（浏览器 console 实抓堆栈）

```
RangeError: Invalid language tag: zhCN
    at Number.toLocaleString
```

本应用的 `i18n.language` 用的是**自有标签**（`zhCN`，无连字符），
`Number.prototype.toLocaleString('zhCN')` 直接抛 RangeError → 渲染期异常 → 错误边界接管整页。
v0.0.8 新增的"内存日志占用（{{rows}} 行）"那行正是 `memoryLogRows.toLocaleString(i18n.language)`。

**为什么测试没拦住**：测试里的 i18n 实例是 `lng: 'en'`（合法标签），所以从不触发。

## 修复

`log-settings-section.tsx`：改为 `toIntlLocale(i18n.language)`（项目既有归一化工具，
`zhCN → zh-CN`，非法值返回 undefined → 退化为默认区域）。全仓扫描确认**仅此一处**裸用；
其余 `i18n.language` 的 Intl 调用原本就走 `toIntlLocale`。

回归测试：`log-settings-memory-usage.test.tsx` 新增用例，用 `lng: 'zhCN'` 的 i18n 实例渲染并断言
行文案正常（修复前必然抛 RangeError）。`renderSection()` 增加可选 i18n 实例参数以支持该用例。

## 验证

- `node -e "(176000).toLocaleString('zhCN')"` → RangeError（修复前必崩）；归一化后 → `176,000`；
- 受影响文件 6/6 pass；全量 **208 pass / 0 fail**；typecheck / lint(0 error) / build / i18n:check / format:check / knip(files) 全绿。

## 发版与真机验证

v0.0.9 发布（镜像 `latest`=`0.0.9` 同摘要；Release 三资产齐备）→ 用户实例重建到 0.0.9 →
真实浏览器打开该页：`crashed: false`，显示 **`内存日志占用 419 Bytes / 上限 200 MB（1 行）`**（顶栏 `New API v0.0.9`）。

## 教训（写进留痕）

1. 语言标签必须走项目归一化工具，**禁止**把 `i18n.language` 直接交给 Intl；
2. 默认 `lng: 'en'` 的测试环境**覆盖不到**真实语言标签路径，涉及 Intl/ToLocale 的组件要用真实标签（如 `zhCN`）测；
3. 交付前应对**被改动的页面**做一次真实浏览器冒烟（本轮正是这样抓到的，而非依赖成员自测）。
