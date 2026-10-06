# 修复F：用户视角找漏 + 新版本可见性（2026-10-06）

范围：`web/**`。基线：`bun test` 154 pass / 0 fail；3020 运行的是**镜像内嵌的前端**（非 `web/dist`）。

---

## 一、用户视角问题清单

### A. 「用户怎么知道有新版本」→ **看不见**

| 调查面 | 实测结果 |
| --- | --- |
| 公开首页（`bw` → `http://127.0.0.1:3020/`） | 正文 1184 字，**无任何版本字样** |
| 关于页（`/about`） | 只有「未设置关于内容」+ 仓库链接 + 页脚归属，**无版本** |
| 登录页（`/sign-in`） | 43 字，**无版本** |
| 面板头部 | 见下：版本行是**死 UI**，永不渲染 |
| 后端 | `/api/status` 只有 `version`（当前版本），**没有** `latest_version` 之类字段 |

**结论**：普通用户（含管理员）在面板里**看不到自己跑的版本**，也没有任何"有新版本"信号。
唯一能看到版本的地方是 `系统设置 → 运维 → 系统维护` 的 "Current version" 卡片（管理员专属），
而那里的"检查更新"按钮是**浏览器直连 `api.github.com`**（见 C）。

**根因（死 UI）**：`system-brand.tsx` 的 `sidebar` 分支（`system-brand.tsx:80-102`，第 97 行渲染版本）
**从未被渲染** —— 全仓唯一调用点是 `app-header.tsx:115 <SystemBrand variant='inline' />`，
而 inline 分支只渲染 logo + 名称；`defaultVersion` prop 也无任何调用方传入。

### B. 面板显示与后端是否一致

- **版本**：面板头部原本不显示；3020 内嵌构建的 `window.__APP_BUILD__` 实测为
  `{"rev":"rv.0000.2k6e8r7p","ch":"2k6e8r7p"}` —— 与真实版本 `0.0.6` 无关（即 Lead① 现象，已复现）。
- **数据库状态/计费概览**：本次未发现不一致（`/api/status` 与 UI 取值同源）。
- **初始化状态**：`/api/status` 的 `setup` 与 `/api/setup` 的 `status` 一致（都是 `false`），
  但**前端表现不一致** → 见 D。

### C. 更新检查：浏览器直连 GitHub（离线/大陆部署必失败）

`update-checker-section.tsx:59-67` 用 `fetch('https://api.github.com/repos/zhemed/new-api-own/releases/latest')`
**从浏览器直接外呼**。自建网关常部署在内网/无外网环境，该按钮必然报错；
且有 60 次/小时未认证限流。**本次未删除该按钮**（属产品决策），仅修掉其比较逻辑；
建议后续改由后端提供 `latest_version` 字段（见"需要后端"一节）。

### D. 初始化守卫被永久绕过（已复现，已修）

- **现象**：同一个服务、同一个路径，`http://127.0.0.1:3020/sign-in` 正常渲染登录页，
  而 `http://10.0.0.91:3020/sign-in` **被重定向到 `/setup`**。
- **根因**：`__root.tsx` 把 `setup_status_checked=true` **持久化进 localStorage**，
  且内存标记 `setupStatusChecked` 直接由它初始化 ⇒ 一旦写过，该浏览器**永远**跳过 setup 检查。
  127.0.0.1 之所以"正常"，正是这条陈旧缓存把守卫跳过了（后端其实 `root_init:false`，根本没有 root 用户）。
- **用户影响**：数据卷被清空/重建/恢复空备份后，用户只会停在登录页并收到"用户名或密码错误"，
  **永远进不去 `/setup`**，且无法自愈（只能手动清站点数据）。
- **复现步骤**：① 任一实例先正常访问一次；② 控制台 `localStorage.setItem('setup_status_checked','true')`；
  ③ 让后端回到未初始化状态（清空数据卷）；④ 刷新 —— 旧代码不会跳 `/setup`。
- **修复**：改用 `sessionStorage`（保留原作者"刷新不重复检查"的意图，把过期窗口收敛到一个会话）。
  **实测**：在开发服务器上写入该 localStorage 陈旧值后刷新，仍正确落到 `/setup`。

### E. 多个复制按钮在 HTTP 部署下失效（已复现，已修）

- **前置事实（实测）**：在本机 `http://10.0.0.91:3020` 上 `window.isSecureContext === false`
  且 `navigator.clipboard === undefined` —— 自建面板通常就是这么访问的。
- **缺陷**：5 处直接调 `navigator.clipboard.writeText(...)`，绕过了项目自带的
  `src/lib/copy-to-clipboard.ts`（该文件注释明确写着"works in HTTP"，用 `execCommand` 兜底）。
  其中最严重的是 `audio-preview-dialog.tsx:123`：**没有 try/catch、没有 await**，
  下一行却直接 `toast.success('Copied')` ⇒ 日志页音频预览的"复制链接"点了**毫无反应也不报错**。
- **复现步骤**：① 用 `http://<局域网IP>:<端口>` 打开面板（非 localhost/HTTPS）；
  ② 进"日志"，打开一条音频/语音日志的预览弹窗；③ 点"复制链接" —— 无提示、无复制。
  对比：渠道页的复制按钮正常（走的是 `useCopyToClipboard` → 有兜底）。
- **修复**：5 处全部改走 `copyToClipboard()` 公共实现（含 `ai-elements/code-block.tsx`）。

---

## 二、落成修改

### Lead①：构建版本恒为常量 → **已修并实测**

- **根因**：`rsbuild.config.ts` 只 `loadEnv()` **收集** `VITE_*`，却没有 `source.define`，
  值从不进入客户端代码；`build-metadata.ts` 于是永远走 `0000` 兜底。
  另外 Rsbuild 自己拥有 `import.meta.env` 对象，**点号式** `'import.meta.env.VITE_X'` define
  在变量不存在时会被它覆盖（实测发射为 `let e;` = undefined）。
- **改法**：`process.env.VITE_REACT_APP_VERSION` 先由 `VERSION` 文件兜底补齐 → `loadEnv` →
  `source.define = { ...env.publicVars, __APP_VERSION__: JSON.stringify(appVersion) }`；
  `build-metadata.ts` 优先读 `__APP_VERSION__`。
- **实测**：
  - `VITE_REACT_APP_VERSION=9.9.9 bun run build` → 产物含 `9.9.9`（Docker 路径有效）；
  - `bun run build` → 产物含 `0.0.6`（仓库 `VERSION` 兜底有效）；
  - 发射代码：`function(){try{return"0.0.6"}...}`；
  - **浏览器实测 before/after**：3020 内嵌旧构建 `rv.0000.2k6e8r7p` → 新代码 `rv.0.0.6.2k6e8r7p`。

### Lead②：更新检查恒报"有新版本" → **已修 + 单测**

- **根因**：`update-checker-section.tsx:78` 用 `data.tag_name ('v0.0.6') === currentVersion ('0.0.6')`
  严格比较，永不相等。
- **改法**：新增 `src/features/system-settings/utils/version-compare.ts`
  （`normalizeVersion` 去 `v`/`V` 前缀与 build metadata；`compareVersions` 返回
  `equal|newer|older|unknown`，逐段数值比较，缺失段按 0，预发布版低于正式版）。
  比较结果驱动 UI：`equal/older` → "已是最新版本" toast；`newer` → 弹"New version available"；
  `unknown`（拿不到当前版本）→ 中性标题 "Release details"，**不再谎报有更新**。
  **纯本地字符串处理，未新增任何网络请求或依赖。**
- **单测** `utils/__tests__/version-compare.test.ts`：**11 pass**，覆盖
  `v` 前缀 / 相等 / 确实更新 / 数值比较（`0.0.9 < 0.0.10`）/ 运行版更新 / 历史 `v0.0.4` / 预发布 / 缺值。

### Lead③：`system-brand.tsx:97` 死 UI → **已修（两者都做）**

- **删**：不可达的 `sidebar` 分支 + 从未被传入的 `defaultName`/`defaultVersion` prop
  （含 `variant` prop，调用点同步改为 `<SystemBrand />`）。
- **加**：把版本真正显示出来 —— 头部品牌药丸内渲染 `data-testid="system-brand-version"`，
  即登录后每个用户都能在顶栏看到当前版本。
- **回归测试** `__tests__/system-brand.test.tsx`：**2 pass**
  （后端报了版本 → 头部显示 `0.0.6`；后端没报 → 不渲染版本块）。

### 顺带：系统维护页显示 Build ID

`update-checker-section.tsx` 的 "Current version" 卡片下新增 `Build ID: rv.0.0.6.2k6e8r7p`，
让 Lead① 的修复在 UI 里可被用户/支持直接读到。新 i18n 键 `Build ID` 已按规范
（临时 `add-missing-keys.mjs` 七语言 + `sync-i18n.mjs`，用完删除）补齐，未手改 locale JSON。

---

## 三、实测结果（全部 exit 0）

| 命令 | exit | 结果 |
| --- | --- | --- |
| `bun run typecheck` | 0 | 无输出 |
| `bun run lint` | 0 | 21 warnings / **0 errors** |
| `bun test` | 0 | **167 pass / 0 fail**（基线 154，新增 13） |
| `bun run build` | 0 | 产物含真实版本 `0.0.6` |
| `bun run i18n:check` | 0 | 4163 引用键七语言齐备 |
| `bun run knip --include files` | 0 | 无输出（0 未使用文件） |

浏览器实测（`bw`）：3020 旧构建 `rv.0000.2k6e8r7p` → 开发服务器新代码 `rv.0.0.6.2k6e8r7p`；
写入陈旧 `setup_status_checked` 后刷新仍正确重定向 `/setup`。验证用的开发服务器已关闭（3020 仍 200）。

---

## 四、未完成 / 需要 Lead 定夺

1. **无法在 3020 上实测"登录后头部显示版本"**（两条硬阻碍）：
   - 3020 跑的是**镜像内嵌的旧前端**，`bun run build` 只更新 `web/dist`，不重新部署就不生效；
   - 3020 本身**尚未初始化**（`/api/setup` → `{"status":false,"root_init":false}`，无 root 用户），
     无法登录，自然也进不了带顶栏的认证布局。
   问：是否允许我（a）在 3020 上完成初始化向导建一个本地管理员 + （b）重建/重启以部署新前端，从而截图取证？
   在获得授权前我没有改动该实例（也未重启任何服务）。
   目前的替代证据：构建产物 grep + 组件回归测试断言 DOM 显示 `0.0.6`。
2. **C 项（浏览器直连 GitHub）未改**：属产品决策。若要"由后端字段驱动"，
   需 Go 侧在 `/api/status` 增加 `latest_version`（例如启动后异步查一次 GitHub releases 并缓存）——
   **需要后端改动，按约束停下报告由 Lead 指派**。前端已具备纯本地的比较能力（`version-compare.ts`），
   后端一旦给出该字段即可直接接入。
3. **D 项为行为变更**：`localStorage → sessionStorage` 会改变初始化守卫的触发频率
   （同一会话内仍只查一次，跨会话会重新校验）。若认为不可接受，回退方式是恢复用 localStorage 读写。
