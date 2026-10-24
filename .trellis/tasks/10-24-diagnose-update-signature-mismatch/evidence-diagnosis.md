# 自更新签名校验失败：只读诊断报告

范围：/root/new-api-own（Trellis 任务 10-24-diagnose-update-signature-mismatch）
性质：全流程只读；未触发任何更新动作、未点任何 POST 路由、未下载二进制大文件、未访问未点名主机
外部证据：GitHub 公开 API 与仓库内工作流文件（只读）

---

## 一、结论速览

用户报告的两个现象是**两个互相独立的问题**，其中一个是真代码缺陷，另一个站不住：

| 编号 | 现象 | 判定 | 一句话 |
|---|---|---|---|
| S1 | 点更新报「更新包的签名与公布的校验值不匹配」 | 已证实（代码缺陷） | 资产白名单漏了 SHA256SUMS.sig，applier 永远拿不到签名资产，必然走 signature_mismatch |
| S2 | 实例已是 v0.0.45，面板却提示更新到 v0.0.45 | 未复现（推断为时序/陈旧页面） | v 前缀比较逻辑正确（已证实）；当前进程自报 v0.0.45 = 最新 tag，服务端判定必为 up_to_date |
| S3 | 发布资产是否真的不一致 | 已排除（已证实） | v0.0.45 四个资产齐全；SHA256SUMS 内容经我复算与官方 digest 逐字节一致 |

要点：

- 【已证实】S1 的根因是**客户端代码缺陷**，不是发布侧问题：`githubReleaseFetcher.FetchLatestRelease` 会用白名单裁剪资产（service/update_check.go:276、:289-299），白名单只有 `SHA256SUMS` 与两个架构二进制，**没有 `SHA256SUMS.sig`**；而 applier 在下载任何东西之前就要求该资产存在（service/update_apply.go:207-212）。两者组合 → 无论 Release 资产多完整，都稳定返回 `signature_mismatch`（HTTP 502）。
- 【已证实】S2 不是「v 前缀/字符串比较」问题：`common.NormalizeVersion` 会剥掉一个前导 v/V（common/version_compare.go:24-32），容器的运行进程自报 `v0.0.45`（容器内执行 `/new-api --version`），最新 Release tag 也是 `v0.0.45` → 比较结果为 equal → 服务端状态 `up_to_date`，面板不显示「有更新」入口（update-checker-section.tsx:103-104）。
- 【推断】用户看到「更新到 v0.0.45」并点击，最可能发生在**当前容器升级到 v0.0.45 之前**（Release 于 2026-10-23 14:25:38Z 发布；本机容器 v0.0.45 进程的启动日志时间为 16:59:25 与 17:03:45），或发生在**另一个实例**上。此推断无法在本轮只读范围内闭环（见第八节）。
- 【已证实】影响面：面板内「立即更新」自 v0.0.43 起**完全不可用**（v0.0.43/44/45 都同时含签名要求与漏项白名单；v0.0.42 及更早没有签名要求）。这是可用性故障，**不是安全退化**——失败方向是 fail closed，拒绝替换是安全的一侧。

---

## 二、验签机制与 signature_mismatch 的全部触发条件

### 2.1 机制（读代码得出）

| 环节 | 事实 | 证据 |
|---|---|---|
| 签名算法 | ed25519 **分离签名**（不是 cosign keyless、不走 OIDC/Sigstore，无身份/issuer 约束） | service/update_apply.go:5、:50-64 |
| 信任根 | 二进制内嵌公钥列表（raw 32 字节标准 base64，也兼容 PEM/PKIX），当前一把 | service/update_apply.go:62-64 |
| 签名对象 | `SHA256SUMS` 文件的**原始字节**，不做归一化；只对 `.sig` 内容做 TrimSpace | service/update_apply.go:312-331、service/update_check.go:37-39 |
| `.sig` 格式 | 一行 base64（标准/无填充/URL 变体）或 hex，解码后必须恰好 64 字节 | service/update_apply.go:334-350 |
| 资产命名 | `SHA256SUMS`、`SHA256SUMS.sig`、`new-api-linux-amd64`、`new-api-linux-arm64` | service/update_check.go:36-45 |
| 体积上限 | SHA256SUMS 64 KiB、签名 8 KiB、二进制 256 MiB | setting/update_check.go:59-66 |
| 校验顺序 | 资产存在性检查 → 下载 SHA256SUMS（取原始字节）→ 下载 .sig → 验签 → 下载二进制并校验 SHA256 → 生成 .prev → 原子替换 | service/update_apply.go:197-247 |
| 失败不留痕 | 验签发生在下载二进制**之前**；T1 分支（缺资产）连临时文件都不会创建 | service/update_apply.go:207-232 |
| 路由与权限 | 三条 update 路由均为 RootAuth（GET 也需 root 会话；系统访问令牌被拒） | router/api-router.go:27-29、controller/update_check.go:26-31 |
| 状态判定 | 运行版本 vs tag 比较 → update_available / up_to_date / unknown | service/update_check.go:195-218 |

### 2.2 `signature_mismatch` 触发条件（穷举）

| 编号 | 条件 | 代码位置 | 是否会记录日志 | 本轮是否命中 |
|---|---|---|---|---|
| T1 | 资产列表（**已裁剪后**）里没有 `SHA256SUMS.sig` 或下载 URL 为空 | service/update_apply.go:207-212 | 否，静默返回 | 是（根因） |
| T2 | 内嵌公钥列表为空（fail closed） | service/update_apply.go:317-319 | 否（由调用方统一记） | 否 |
| T3 | `.sig` 内容无法解码为 64 字节（base64/hex 全失败） | service/update_apply.go:333-350 | 是，经 :224 记 SysError | 否（到不了） |
| T4 | 签名无法被任何内嵌公钥验证通过 | service/update_apply.go:320-330 | 是，经 :224 记 SysError | 否（到不了） |
| T5 | 公钥本身不可解析（逐个跳过，全部不可解析则等同 T4） | service/update_apply.go:321-325 | 是（逐把记 SysError） | 否 |

补充（不产生 signature_mismatch 的相邻失败）：`.sig` 下载失败走 `checksum_download_failed`（service/update_apply.go:581），前端归类为 download_failed；`SHA256SUMS` 里查不到目标资产行走 `checksum_missing`（:303-305）；`SHA256SUMS` 资产整体缺失走 `checksum_asset_missing`（:202-206）。因此**用户看到的那句文案只可能来自 T1–T5**（web/src/features/system-settings/maintenance/update-api.ts:123-125 把 signature_mismatch / signature_invalid / signature_verify_failed 都映射到同一文案；zh.json:4358）。

### 2.3 关键代码缺陷（白名单漏项）

```
service/update_check.go:276      release.Assets = filterUpdateAssets(release.Assets)
service/update_check.go:289-299  kept := make([]ReleaseAsset, 0, 3)
                                 if asset.Name == UpdateAssetNameSHA256Sums ||
                                    asset.Name == UpdateAssetNameForArch("amd64") ||
                                    asset.Name == UpdateAssetNameForArch("arm64") { ... }
```

- 裁剪发生在 **check 与 apply 共用的 fetcher** 内（`defaultUpdateSource()`，service/update_check.go:331-345；applier 用它构造，service/update_apply.go:661-673），所以 apply 路径看到的也是裁剪后的列表 → T1 必然命中。
- 该组合自 v0.0.43 引入（提交 813d5c4「自升级权限收紧/资产签名/不持久提示/回滚点」）：`git show v0.0.42` 无签名要求，v0.0.43/44/45 三个 tag 的白名单都是同一份漏项实现。
- 现有单测抓不到：`update_apply_test.go` 直接构造 `ReleaseInfo`（绕开 fetcher 的裁剪，service/update_apply_test.go:58-78）；fetcher 测试的 payload 里根本没有 `SHA256SUMS.sig`（service/update_check_test.go:335-345），也没有任何测试断言裁剪后仍保留 applier 需要的资产。

---

## 三、发布侧证据（GitHub 公开只读）

### 3.1 v0.0.45 资产清单与 v0.0.44 对照

| 资产 | v0.0.45 大小 | v0.0.45 digest（sha256） | v0.0.44 大小 | v0.0.44 digest（sha256） |
|---|---|---|---|---|
| new-api-linux-amd64 | 118513848 | 4e8546fe680d95bdd797c32913d1f4741a52e3f68ea690aa447746e57449c9a7 | 118513848 | 60742dcf1199409c182f9a9c9f48e5b7eeaed43c6b0a2d6bad0eab8b6b10986e |
| new-api-linux-arm64 | 115474616 | 37daf69caab462fc3e7ba1f3b656af727b04d2e9e55b13c179e8dd48f427911e | 115474616 | e0af57e3226fd7d9560cdb7848e3a70b3a9e01403923f733c4da312c87123d48 |
| SHA256SUMS | 172 | 7125fac85e2d74b5a28a4cc841b6ad12837660596c32c7c6b52b8d698c33d12a | 172 | 318b681aaacbd6de5a8609ebb36ad946e1f213f5646f5ab02f18f65b38eb24c3 |
| SHA256SUMS.sig | 89 | 11a91894f4ff0575292aead67e2fa691e19deea022c15462b97783174d096112 | 89 | 81912b2f55213d93f7cc553cc4a1f160a7a917572e84b1ac874732c1d457a18c |

结论：【已证实】两个版本的**四个资产都齐全**，`.sig` 存在且大小符合「一行 base64 的 64 字节签名 + 换行」（64 字节 base64 = 88 字符 + 换行 = 89）。发布侧没有「缺签名资产」的问题。

### 3.2 SHA256SUMS 内容一致性（我独立复算，未下载大文件）

用 GitHub 元数据里两个二进制的 digest 重建 `sha256sum` 格式的两行文本，再与官方公布的 SHA256SUMS digest 对比：

| 版本 | 重建字节数 | 重建内容 sha256 | 官方 digest | 一致 |
|---|---|---|---|---|
| v0.0.45 | 172 | 7125fac85e2d74b5a28a4cc841b6ad12837660596c32c7c6b52b8d698c33d12a | 7125fac85e2d74b5a28a4cc841b6ad12837660596c32c7c6b52b8d698c33d12a | 是 |
| v0.0.44 | 172 | 318b681aaacbd6de5a8609ebb36ad946e1f213f5646f5ab02f18f65b38eb24c3 | 318b681aaacbd6de5a8609ebb36ad946e1f213f5646f5ab02f18f65b38eb24c3 | 是 |

结论：【已证实】`SHA256SUMS` 恰好是 `<amd64 digest>  new-api-linux-amd64` 与 `<arm64 digest>  new-api-linux-arm64` 两行加尾换行（172 字节），**与官方为二进制公布的 digest 逐字节一致**，即校验和文件正确覆盖了两个二进制。发布侧不存在「校验值与二进制对不上」。

### 3.3 签名是怎么产生的，有没有「签后重传/重跑」

| 环节 | 事实 | 证据 |
|---|---|---|
| 版本对齐 | 工作流把 tag 原样写进 VERSION（含前导 v），使镜像与二进制都带 v | .github/workflows/release.yml:127-133、docker-build.yml:56-76 |
| 二进制构建 | `go build -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=<VERSION 文件内容>"`（编译期注入版本） | release.yml:160-163、Dockerfile:29 |
| 生成校验和 | `sha256sum new-api-linux-amd64 new-api-linux-arm64 > SHA256SUMS` | release.yml:168-173 |
| 签名 | openssl ed25519：`pkeyutl -sign -rawin -inkey key.pem -in SHA256SUMS -out sig.bin`；**发布前自校验** `-verify -rawin`；再 `base64 -w0` + 单个换行写成 `SHA256SUMS.sig` | release.yml:176-225 |
| 上传 | 一次性上传四个资产：两个二进制 + SHA256SUMS + SHA256SUMS.sig | release.yml:235-236 |
| 上传后校验 | 工作流回查四个资产是否齐全，并额外检查有没有多余资产 | release.yml:286-296 |
| 私钥来源 | `UPDATE_SIGNING_KEY` secret；未配置时签名步骤**主动失败**，不会发布未签名 Release | release.yml:188-199 |

【已证实】资产时间线无「签后重传」迹象：v0.0.45 的 created_at 14:22:16Z、published_at 14:25:38Z，四个资产的 created_at 均为 14:25:34Z；`SHA256SUMS` 与 `SHA256SUMS.sig` 的 updated_at 都是 14:25:34Z（即上传后未被覆盖），只有两个大二进制在 14:25:34→14:25:38 之间处于上传完成过程中。v0.0.44 同样是 12:49:31Z 一次性上传。

---

## 四、实例侧证据（只读）

| 检查 | 结果 | 命令或路径 |
|---|---|---|
| 容器与镜像 | 容器 new-api 运行中，镜像 ghcr.io/zhemed/new-api-own:v0.0.45，Up 8 hours | docker ps --filter name=new-api |
| 运行进程自报版本 | **v0.0.45** | docker exec new-api /new-api --version |
| 启动横幅（进程自己打的） | `New API v0.0.45 started`，时间戳 2026/10/23 - 16:59:25（另一条 17:03:45） | docker logs new-api；来源 main.go:61 打印 common.Version |
| 进程版本是否可能被环境变量覆盖 | 否：容器 Env 里没有 VERSION（只有 PATH 与日志相关变量），故 ldflags 注入值有效 | docker inspect new-api --format '{{json .Config.Env}}' |
| 更新相关日志 | 只有启动时的 update check 配置行，**没有任何 update apply / signature / not_newer / checksum 报错行** | docker logs new-api |
| /data 下的更新产物 | 无 `.prev`、无 `.new-api-update-` 前缀的临时下载文件、无 SHA256SUMS 落盘 | find data -maxdepth 3 |
| 容器内 /new-api.prev | 不存在（说明本容器从未成功执行过 apply；apply 成功前必先写 .prev） | docker exec new-api ls -la /new-api.prev |
| 容器内 /new-api 哈希 | 278dbfc97cc08feb15d423079317ea1f3ada231f2eba91d49b08057f89f989ca，与官方 amd64 资产 digest 不同 → 运行的是镜像内构建产物，不是下载回来的发布资产 | docker exec new-api sha256sum /new-api |
| 更新检查缓存是否落库 | 不落库：options 表共 14 个键，无任何 update/version 相关键（缓存在进程内存，TTL 1h） | python3 sqlite3 只读打开 data/one-api.db；setting/update_check.go:54-55 |
| 面板入口条件 | 仅当服务端 state=update_available 才显示「新版本可用」与更新按钮；not_newer 有独立文案「This build already matches that version.」 | update-checker-section.tsx:103-104、:125-128、:170-171 |
| 文案对应 | 「更新包的签名与公布的校验值不匹配，因此已拒绝替换。」= signature_mismatch 专用文案 | web/src/i18n/locales/zh.json:4358；update-checker-section.tsx:180-183 |

时间线（都是仓库内可查的事实）：

1. 2026-10-23 12:49:35Z：v0.0.44 发布。
2. 2026-10-23 14:25:38Z：v0.0.45 发布（四个资产齐全）。
3. 本机容器日志显示 v0.0.45 进程在 16:59:25 与 17:03:45 启动（对应 session 记录里的容器重建与重启存活验证，.trellis/workspace/new-api-own/journal-2.md:1247、:1259）。
4. 会话记录写明本地实例是「用镜像方式升到 v0.0.45」，并把旧容器留作回滚点（journal-2.md:1115、:1123）。

【推断】14:25（v0.0.45 发布）→ 16:59（本机升级到 v0.0.45）之间存在约 2.5 小时窗口，此时实例仍跑旧版本，面板**合法地**提示「更新到 v0.0.45」；用户在该窗口内点击就会走到 apply 路径并命中 T1。

---

## 五、根因判定（证据链）

【已证实】S1 的完整证据链：

1. 面板点更新 → POST /api/status/update/apply（RootAuth）→ controller.ApplyUpdate（router/api-router.go:28、controller/update_check.go:33-50）。
2. applier 用与 check 共用的 fetcher 拉取 `/releases/latest`（service/update_apply.go:183-189），该 fetcher **必然**裁剪资产（service/update_check.go:276）。
3. 裁剪白名单只保留 `SHA256SUMS` 与两个二进制（service/update_check.go:289-299），**丢弃 `SHA256SUMS.sig`**。
4. applier 在下载任何东西前要求 `SHA256SUMS.sig` 存在（service/update_apply.go:207-212）→ 条件必然为假。
5. 于是返回 `code=signature_mismatch`、HTTP 502、message「release has no SHA256SUMS.sig; refusing to replace the binary」（同上）。
6. 前端把该 code 映射为 signature_mismatch 文案（update-api.ts:123），中文即用户看到的那句（zh.json:4358）。
7. 与实例侧观测吻合：没有任何 apply 报错日志（T1 分支静默）、没有任何临时产物或 .prev、/data 干净。
8. 与发布侧证据吻合：GitHub 上 `.sig` 明明存在（第三节），排除「发布资产缺失」。

【已证实】S2 的排除证据链：

1. 运行进程自报 v0.0.45（容器内 `/new-api --version`；启动横幅 main.go:61）。
2. 最新 Release tag = v0.0.45（GitHub API）。
3. 比较逻辑：`NormalizeVersion` 剥 v/V 前缀与构建元数据（common/version_compare.go:24-32），逐段数值比较（:37-69）→ v0.0.45 vs v0.0.45 = equal。
4. equal → 服务端 state=up_to_date、has_update=false（service/update_check.go:207-217）。
5. 前端以服务端 state 优先（update-status.ts:53-75），up_to_date → 不显示更新按钮（update-checker-section.tsx:103-104）。
6. 结论：**当前构建 + 当前最新版本不会出现「已是最新却提示更新」**；v 前缀不是原因（其归一化在服务端与前端各有一份且语义一致）。

【推断待验证】用户实际点击的那一次，最可能是「升级前的旧构建」或「另一实例」：这两条都能解释「面板提示更新到 v0.0.45」（服务端确实算出 update_available），也能解释随后命中 T1。当前本地容器无法复现该提示（见上）。

---

## 六、影响面

| 维度 | 结论 |
|---|---|
| 是否阻断正常升级 | **阻断面板内自更新**：v0.0.43 / v0.0.44 / v0.0.45 三个版本里，任何一次「立即更新」都会失败（HTTP 502 signature_mismatch），失败点在下载之前，不会损坏现有二进制。v0.0.42 及更早无签名要求，不受影响。 |
| 是否影响镜像升级路线 | 不影响。镜像方式升级（docker run 新 tag、重建容器）与验签无关；本次实例就是靠镜像路线升到 v0.0.45 的。 |
| 是否有安全含义 | 无安全退化：这是 fail closed 的误判——系统拒绝了**本来合法**的升级，而不是接受了伪造的升级；未签名/被篡改的资产依旧无法通过（T2–T5 仍然有效）。 |
| 次要风险 | ① 文案误导：把「资产缺失」说成「签名与校验值不匹配」，会把排查引向发布侧/密钥，实际二者都正常；② T1 分支不记日志，现场无痕迹（对比 T3–T5 会记 SysError）；③ 契约漂移无人看守：白名单与 applier 需求分处两个文件，且现有测试都绕开这条真实路径。 |
| 回滚/演练影响 | 会话记录里的回滚演练依赖 apply 成功产生的 `.prev`（controller/update_check.go:33-50、service/update_apply.go:234-240）；自更新不可用 → 面板内的更新与回滚演练都无法在本机跑通。 |

---

## 七、候选修复方向（本轮不实施，供裁定）

方向 A（最小改动，推荐先做）：把 `SHA256SUMS.sig` 加回白名单。

- 位置：service/update_check.go:289-299（同时把容量 3 改成 4，纯清理）。
- 效果：T1 消失，验签真正执行；若发布侧签名字节正常，apply 即可走完下载→校验→.prev→替换。
- 风险：低。该白名单只影响"带进内存与响应的资产"，加一项不改变任何安全判断。

方向 B（结构性，消除同类漂移）：不要在共用 fetcher 里做裁剪。

- 方案：fetcher 返回全量资产；只在"状态响应组装"处裁剪（`filterUpdateAssets` 移到 GetUpdateStatus/currentStatus 边界），或在 applier 侧单独取用不裁剪的资产视图。
- 效果：从根上避免"applier 需要的资产被 check 侧白名单裁掉"这一类缺陷再犯（本轮缺陷正是这个类别）。
- 风险：中。要确认状态响应不把整份资产列表暴露给前端（当前裁剪的部分动机就是控制响应体积，service/update_check.go:288 注释），需要保留响应侧裁剪。

方向 C（可诊断性与文案准确性）：让失败更可辨认。

- ① T1 分支补一行 SysError（记录缺失的资产名与本次可用的资产名），与 T3–T5 的日志待遇一致；
- ② 给"签名资产缺失"单独的稳定错误码（例如 signature_asset_missing），前端给专用文案，不再复用"签名与公布的校验值不匹配"；
- ③ 回归保护：新增"fetcher → applier"集成测试，用真实四资产 payload 断言 apply 能走到验签并通过（当前测试都绕开 fetcher，service/update_apply_test.go:58-78、service/update_check_test.go:335-345），可选再加一条"applier 需要的每个资产名都在白名单里"的静态/表驱动断言。
- 风险：低，但属行为可见变化（新增错误码需前后端同步）；建议与 A 一起做。

补充（版本侧）：S2 不需要修复；如仍想减少"陈旧页面显示有更新"的困惑，可考虑面板在切回标签页/打开时刷新状态，属体验优化、优先级低。

---

## 八、不确定项与异常（含无法闭环的部分）

1. **无法确认用户点击的是哪一个实例**：本轮只读且只允许访问 GitHub 公开页面，没有向任何实例发请求；另一个（生产）实例的版本与面板状态未经核实（访问未点名主机需另行授权）。若该实例仍 ≤ v0.0.44，则「提示更新到 v0.0.45 + 点击报签名错」在那里是**完全可复现**的；若确认用户点的是本机面板，则点击必然发生在 16:59 之前（旧构建）。
2. **`SHA256SUMS.sig` 的字节内容未能取回**：GitHub 资产下载会 302 到 release-assets 主机，只读抓取工具按设计不跟随跨域跳转；因此我没有独立用内嵌公钥复算「签名是否有效」。这一点对本次结论**不构成影响**——T1 在下载 `.sig` 之前就已拒绝；且发布侧有 CI 自校验（release.yml:214-225）。若要补做，见第九节的 openssl 命令（需另行授权下载 89 字节文本）。
3. **apply 尝试的日志证据缺失**：本机日志里没有任何 apply 记录；但 T1 分支本身不写日志（service/update_apply.go:207-212），且失败可能发生在已被重建掉的旧容器上，因此「无日志」同时兼容两种解释，不能用来定位。
4. **options 表无更新缓存**：更新检查结果只存在进程内存（TTL 1 小时，service/update_check.go:139-145、setting/update_check.go:54-55），数据库里没有可回溯的记录，故无法从库里还原当时的判定。
5. **容器内 /new-api 哈希与发布资产不同**（278dbfc9… vs 4e8546fe…）：这与"容器用的是镜像内构建产物、从未成功做过下载式自更新"一致；但构建环境不同也可能导致同源不同哈希，故不作为独立结论，仅作旁证（另有 `.prev` 不存在这一更强的旁证）。
6. **未发现状态待修复类异常**：任务目录 task.json 可读；归档与活动任务状态无冲突（此项与本轮问题无关，仅说明本轮未见数据层异常）。

---

## 九、复核命令与越界自述

仓库内只读命令（本轮实际执行）：

```bash
# 代码路径
sed -n '260,300p' service/update_check.go
sed -n '190,260p' service/update_apply.go
sed -n '300,380p' service/update_apply.go
cat controller/update_check.go
sed -n '20,70p' setting/update_check.go
sed -n '20,70p' common/version_compare.go
grep -n "status/update" router/api-router.go
sed -n '55,65p' main.go
sed -n '120,300p' .github/workflows/release.yml
sed -n '50,100p' .github/workflows/docker-build.yml
grep -n "VERSION" Dockerfile
# 发布历史与缺陷引入点
git show v0.0.42:service/update_check.go | grep -c UpdateAssetNameSHA256SumsSig
git show v0.0.43:service/update_check.go | sed -n '/filterUpdateAssets 只保留/,/^}/p'
git log --oneline -6 -- service/update_apply.go
# 前端契约
grep -n "signature_mismatch\|not_newer" web/src/features/system-settings/maintenance/update-api.ts
sed -n '95,140p' web/src/features/system-settings/maintenance/update-checker-section.tsx
sed -n '170,190p' web/src/features/system-settings/maintenance/update-checker-section.tsx
sed -n '4355,4360p' web/src/i18n/locales/zh.json
```

实例侧只读命令：

```bash
docker ps --filter name=new-api --format '{{.Names}} | {{.Image}} | {{.Status}}'
docker logs new-api | grep -iE "update|signature|version"
docker exec new-api /new-api --version
docker exec new-api ls -la /new-api /new-api.prev
docker exec new-api sha256sum /new-api
docker inspect new-api --format '{{json .Config.Env}}'
find data -maxdepth 3 -name '.new-api-update-*'
find data -maxdepth 3 -name '*.prev'
python3 -c "import sqlite3;con=sqlite3.connect('file:data/one-api.db?mode=ro',uri=True);print(con.execute(\"select count(*) from options\").fetchone())"
```

复算与一致性检查（本地计算，无外呼）：

```bash
python3 - <<'PY'   # 用官方 asset digest 重建 SHA256SUMS 并比对官方 SHA256SUMS digest
import hashlib
amd="4e8546fe680d95bdd797c32913d1f4741a52e3f68ea690aa447746e57449c9a7"
arm="37daf69caab462fc3e7ba1f3b656af727b04d2e9e55b13c179e8dd48f427911e"
print(hashlib.sha256(f"{amd}  new-api-linux-amd64\n{arm}  new-api-linux-arm64\n".encode()).hexdigest())
PY
```

外部只读检索（公开主机、无凭据）：

```bash
# 以 web_fetch 执行，等价于：
#   GET https://api.github.com/repos/zhemed/new-api-own/releases/tags/v0.0.45
#   GET https://api.github.com/repos/zhemed/new-api-own/releases/tags/v0.0.44
# 说明：SHA256SUMS 的内容未下载，改为用官方资产 digest 本地复算（第三节 3.2）
```

若要补做签名有效性验证（需另行授权下载 89 字节文本，本轮未做）：

```bash
base64 -d SHA256SUMS.sig > SHA256SUMS.sig.bin
openssl pkeyutl -verify -rawin -pubin -inkey pub.pem -in SHA256SUMS -sigfile SHA256SUMS.sig.bin
# 公钥按内嵌常量还原为 raw 32 字节 ed25519（service/update_apply.go:62-64），步骤见 docs/release-signing.md:87-89
```

越界自述：本轮未修改任何代码、配置、服务、数据库；未执行 git 写操作；未触发任何更新/回滚动作或 POST 路由；未下载二进制大文件；未安装任何软件；未访问未点名主机（仅 GitHub 公开 API）；未读取工作区外文件。唯一写入为本文件 evidence-diagnosis.md。

---

## 附：一句话给发布/维护者

面板自更新自 v0.0.43 起 100% 失败，原因是「check 侧资产白名单漏了 `SHA256SUMS.sig`，而 apply 侧在下载前强制要求它」——一行白名单改动即可修复（方向 A），发布资产与签名本身没有问题；同时建议补上 fetcher→applier 的集成回归与更准确的错误码/文案（方向 C），避免下一次再把「资产缺失」误报成「签名不匹配」。
