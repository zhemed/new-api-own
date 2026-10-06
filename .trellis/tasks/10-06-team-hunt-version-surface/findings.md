# 调查B：版本呈现链（只读实测，2026-10-06 13:40）

环境：本机 3020 演示实例 = `.local-instance/new-api`（pid 476111，13:22:03 启动，二进制 13:20:44 构建）。
仓库 `VERSION` = `0.0.6`；本地镜像 `ghcr.io/zhemed/new-api-own:{latest,0.0.6}` = a74d0a6c0940（1 小时前）。

## 结论（先说）

1. **能让用户看到 0.5 的位置只有「实例本身没更新」**——所有 UI/API 的版本号都取自运行中二进制/镜像的
   `common.Version`，不换镜像就永远显示旧号（`MAINTENANCE.md:248` 记录了本机踩过的原坑：实例停在 `0.0.5`、
   本地 `latest` 还陈旧指向 `0.0.3`）。
2. 次要路径：浏览器 `localStorage['status']` 作为 react-query 占位数据（`use-status.ts:70`），
   旧实例写入的版本会先渲染，请求失败时长期停留。
3. **前端自带的构建版本号是坏的**：实测恒为 `rv.0000.2k6e8r7p`（`VITE_REACT_APP_VERSION` 从未注入），
   不是 0.5，但任何用构建版本核对发版的动作都会得到错误结论。
4. 浏览器侧**没有** SW/陈旧构建问题：3020 实际加载的 `index.5bbdf74f4d.js` 与本仓库 `web/dist` 同文件
   （md5 一致），无 Service Worker。

## 版本呈现位置 · 0.0.6 对照表

| # | 位置 | 期望 | 实测 | 证据 |
|---|---|---|---|---|
| 1 | 公开页（首页/登录） | 不显示 | 不显示 | 3020 页面文本无 `0.0.x`；`app-header.tsx:115` 用 `variant='inline'`（不渲染版本） |
| 2 | 侧边栏品牌版本行 | 0.0.6 | **无从显示（死 UI）** | `system-brand.tsx:97` 仅 sidebar 变体渲染，全仓库唯一调用点 `app-header.tsx:115` 是 inline |
| 3 | 设置→运营→「Current version」 | 0.0.6 | 0.0.6（推断自 API） | `operations/index.tsx:72` 传 `status?.version` → `update-checker-section.tsx:115` |
| 4 | 系统信息→实例面板 `runtime.version` | 0.0.6 | 0.0.6（单实例） | `system-instances-panel.tsx:423` |
| 5 | `/api/status` | 0.0.6 | `"version":"0.0.6"` | 本机 curl |
| 6 | 响应头 `X-New-Api-Version` | 0.0.6 | 0.0.6（所有响应，含 404） | `middleware/cors.go:24` + curl |
| 7 | 启动日志 | `New API 0.0.6 started` | 一致 | `main.go:61`；`.local-instance/app.log:13:22:04` |
| 8 | 镜像 OCI label | v0.0.6 | `org.opencontainers.image.version=v0.0.6` | `docker image inspect`；旧镜像另存 `v0.0.4` |
| 9 | 镜像内 `/VERSION` | — | **不存在**（静态判定） | `Dockerfile:8` 只把 VERSION 放 builder 的 `/build/VERSION`；最终阶段只 COPY `new-api`+licenses；`.local-instance/` 也无 VERSION |
| 10 | 前端构建版本面 | `rv.0.0.6.…` | **`rv.0000.2k6e8r7p`** | 3020 `window.__APP_BUILD__`、`<html data-build-rev>`、`meta[name=build-id]`、`localStorage['app:rev']` |
| 11 | 前端包版本 | — | `web/package.json` = `1.0.0`（从不参与展示） | 文件 |
| 12 | 文档/徽章 | — | 只有发版示例 `0.0.3/0.0.4` | `README.md:61`、`MAINTENANCE.md:378+` |

### 10 号位的根因

`Dockerfile:9`（与 `makefile:13`）确实设置了 `VITE_REACT_APP_VERSION`，但 `web/rsbuild.config.ts` 只把
`loadEnv` 用于 `VITE_REACT_APP_SERVER_URL`（dev proxy），**没有 `source.define` / `publicVars` 转发**，
所以 `web/src/lib/build-metadata.ts:68` 的 `import.meta.env.VITE_REACT_APP_VERSION` 恒为 `undefined`
→ `computeBuildRevision()` 回退 `'0000'`。旁证：3020 的 bundle 里搜不到任何 `"0.0.x"` 字面量，
只有 `BUILD_CHANNEL_TAG`（`g="2k6e8r7p"`）。

## 浏览器缓存 / Service Worker 实测（3020）

- 无 SW：仓库无 `serviceWorker` 代码；实测 `navigator.serviceWorker.controller === false`。
- 页面加载资源：`index.5bbdf74f4d.js`、`index.27870ff397.css`（from `performance.getEntriesByType`）。
- 与工作区 `web/dist/static/js/index.5bbdf74f4d.js` **md5 相同**（`039f3b144c353505ce99e75b2a37c9fc`，
  3434779 字节）→ 运行实例的前端 = 当前工作区构建，**没有停在旧构建**。
- 缓存头：`/` → `Cache-Control: no-cache`；哈希资源 → `max-age=604800`（文件名内容哈希，可接受）。
- `middleware/cache.go:14` 的 `Cache-Version` 是**硬编码常量**（`b688f2fb…`），跨构建永不变化
  → 不具备缓存失效/版本识别能力；前端 `web/src/lib/frontend-cache.ts:19` 用的是自己的 `'default-v1'`，
  与响应头无关（实测 `localStorage['newapi:default:cache-version'] = 'default-v1'`）。
- `/api/status` 无 `Cache-Control`：`middleware.Cache()` 在 `web-router.go:27`，而 API 路由先注册
  （`router/main.go:16` vs `:26`），中间件不作用于 API → 版本号不受 HTTP 缓存影响。

## 会显示旧号的位置（按可能性排序）

1. **运行实例未更新**（主因）：`common.Version` 由构建期 ldflags 决定（`Dockerfile:29`）；
   镜像不换/未 `docker pull`（本地 `latest` 可陈旧）→ UI、API、`X-New-Api-Version`、启动日志一致地显示旧号。
2. **localStorage 占位**：`use-status.ts:70` `placeholderData: getInitialStatus()`；旧实例写下的 `status`
   里带着旧 `version`，首屏先渲染旧号；请求失败时（离线/后端挂）长期停留。
3. **实例面板**：`system-instances-panel.tsx:423` 展示其它节点的 `runtime.version`，旧节点即旧号。
4. **误报「有新版本」**：`update-checker-section.tsx:78` 用 `data.tag_name`（约定带 `v`：`v0.0.6`，见
   `MAINTENANCE.md:407`）与 `currentVersion`（后端 `0.0.6`，无 `v`）严格比较 → 恒不相等，即使已在 0.0.6
   也会弹 `New version available: v0.0.6`。（需联网核对 GitHub tag，本次离线，按约定推断。）
5. 前端构建版本面显示 `0000`（不会显示 0.5，但据此核对版本会得出错误结论）。

## 无法验证

- registry 远端 `latest` 摘要（无外部网络；注意到其他成员在跑 `docker buildx imagetools inspect`，非本任务）。
- GitHub release `tag_name` 实际值（离线，按 `MAINTENANCE.md:407` 的「VERSION 不带 v / tag 带 v」约定推断）。
- 最终镜像内 `/VERSION` 的运行时确认（未启动容器，按 Dockerfile 静态判定）。
