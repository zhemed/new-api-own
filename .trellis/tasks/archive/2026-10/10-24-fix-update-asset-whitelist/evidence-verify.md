# 自更新白名单修复：独立验证证据（verifier）

- 任务：共享任务板 `task-21`（Trellis 任务 `10-24-fix-update-asset-whitelist` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 01:56 起）
- 基线：`HEAD = bce3462`；工作区未提交改动 `15 files changed, 193 insertions(+), 20 deletions(-)`
- 基线指纹：`git diff | sha256sum = aff2a94911e60b26b173b7e1044e92c4db386c41916b3d39475ca71273e79fd4`
- 被验证文件 blob 哈希（结论绑定此版本；任一变化需重跑）：

| 文件 | blob |
|---|---|
| service/update_apply.go | 9f2fe0ec595f07a5122413c0b6a80cb8a7e6f750 |
| service/update_check.go | 84d9c102b90c36c6129fadcf6c7fe3ff1311b127 |
| service/update_apply_test.go | 3faf37fc6dabca61642c8709277ac0173f87482f |
| service/update_check_test.go | be48cdf773065795b9da8b231b8a1b079a24900c |
| web .../maintenance/update-api.ts | 57a44c5fd66db400a3087e61d8e32bad387af1d1 |
| web .../maintenance/update-checker-section.tsx | 0681757c1e0ed82dcb708409cfac4d1225634b25 |
| web .../__tests__/update-api.test.ts | 7fd04f778dba82c3446a3099040fb75ea493e713 |
| web .../__tests__/update-checker-section.test.tsx | e338593aca506a2dec9cb2563a6d5eb28b5bcf2e |
| web/src/i18n/locales/en.json | 888dcbe5f95486eee3d248439b2d50075b1cc008 |
| web/src/i18n/locales/zh.json | 4dbd7a432bf67126aa28579d004d4eb205a8f7c8 |
| web/src/i18n/locales/zh-TW.json | ff84b84e875fe8f1a1464ae02d430a2a289c27d8 |
| web/src/i18n/locales/fr.json | 4c41fe7d5294dc9aa3bb87e1f5c43f52105808fc |
| web/src/i18n/locales/ja.json | 37ad54e7d203cfc06690af013b6f6c32ff8b2b9c |
| web/src/i18n/locales/ru.json | 9e97d0e1d7fc43a46b50e9078f595bd0b8ee6621 |
| web/src/i18n/locales/vi.json | ae34e171442876302f4b75efa3c35f0238486470 |

## 结论

**接受（Accept）**。缺陷修复成立、回归测试确实能抓住该 bug（修复前失败 / 修复后通过，Go 与前端都做了前后对照）、
门禁全绿、响应结构与安全语义未变、七语言齐备无重排、工作区无残留未跟踪文件。
另有 3 条非阻断观察与 2 条不确定项（见末节），均不构成打回理由。

## 一、修复前后对照（关键证据）

方法：`git clone /root/new-api-own /tmp/verify-repo`（改动未提交，克隆即**修复前**代码），
把工作区的测试文件拷进克隆后运行；**全程未在真实工作区做任何还原操作**（未用 checkout/stash/restore）。

### 1.1 Go 侧

首次直接拷测试跑出的不是行为失败而是**编译失败**（`undefined: UpdateApplyCodeSignatureAssetMissing`），
这只能证明测试引用了新常量，不能证明它能抓到缺陷。因此补了一个**仅定义常量、不含任何逻辑**的 shim
（`/tmp/verify-repo/service/zz_prefix_shim_test.go`，只存在于 /tmp 克隆内），让测试能针对**修复前的生产逻辑**编译，
从而得到真正的行为对照：

| 用例 | 修复前（克隆 + shim） | 修复后（真实工作区） |
|---|---|---|
| TestGithubReleaseFetcherParsesStableRelease | FAIL：资产名列表不符（期望 4 项含 `SHA256SUMS.sig`，实得 2 项） | PASS |
| TestApplyUpdateRejectsMissingSignatureAsset（T1） | FAIL：错误码 `signature_mismatch` != `signature_asset_missing` | PASS |
| TestApplyUpdateMissingSignatureAssetReportsCodeAndLogsAssetName | FAIL：错误码不符（同 T1） | PASS |
| TestApplyUpdateSeesSignatureAssetFromRealFetcher（fetcher→applier 直通） | FAIL：`[]string(nil)` 不含 `.../assets/SHA256SUMS.sig` —— applier 连一次下载请求都没发出 | PASS |
| 同批其余 7 个签名/抓取用例 | PASS（未受影响） | PASS（共 11 个用例全绿） |

修复前失败信息摘录：`Error: []string(nil) does not contain "http://127.0.0.1:46443/assets/SHA256SUMS.sig"`
——这条正是根因的可执行复现：旧 fetcher 把 `.sig` 裁掉，applier 在资产检查处直接返回。

### 1.2 前端侧

`/tmp/verify-repo/web` 指回工作区 `node_modules`（符号链接，只读依赖），拷入两个新测试文件后运行：

| 用例 | 修复前 | 修复后 |
|---|---|---|
| update checker apply flow > explains a missing signature asset instead of a signature mismatch | FAIL | PASS |
| applyUpdate > classifies the root-only and signature refusals | FAIL | PASS |
| 两个文件其余用例 | PASS（45） | PASS（47 个全绿，0 fail） |

即前端两条新断言同样是「修复前失败、修复后通过」，不是空转测试。

## 二、门禁复跑（真实工作区）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `GOPROXY=off go build ./...` | rc=0 |
| Go 静态检查 | `GOPROXY=off go vet ./...` | rc=0，无输出 |
| Go 测试 | `GOPROXY=off go test ./service/... -count=1` | ok service 0.456s / ok service/authz / passkey 无测试 |
| 目标用例明细 | `go test ./service/ -run SignatureAsset -count=1 -v`（同形式另跑 Signature、Fetcher 两前缀，合并正则见第八节） | 11 个用例全 PASS |
| 端点用例 | `go test ./controller/ -run 'Update' -count=1` | rc=0 |
| 前端类型 | `cd web && bun run typecheck` | rc=0，无诊断 |
| 前端 i18n | `cd web && bun run i18n:check` | rc=0；引用键 4186 → 4187（+1，与新增一键一致），7 locale 齐备 |
| 前端测试 | `cd web && bun test src/features/system-settings/maintenance/__tests__/` | 70 pass / 0 fail（5 文件） |

Go 门禁全部以 `GOPROXY=off` 运行，杜绝任何模块下载（不触网）。

## 三、逐项代码核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| fetcher 不再裁剪 | `service/update_check.go:278-282` 已删掉 `release.Assets = filterUpdateAssets(...)`；`filterUpdateAssets` 全仓库零残留（Go/TS/MD 全部 grep 无命中，仅任务 PRD 作为历史背景提及） | 通过 |
| applier 取 `.sig` 可达 | `service/update_apply.go:210` `findUpdateAsset(release.Assets, UpdateAssetNameSHA256SumsSig)`；`findUpdateAsset` 为**精确名相等**（`:649-656`），不受完整资产列表顺序影响；直通用例实测 applier 真的请求了 `.sig` URL | 通过 |
| 响应结构一行未改 | `UpdateCheckStatus` 结构体 HEAD 与工作区逐字节相同（`diff` 输出为空）；该结构无 assets 字段；`service/update_check.go` 变更仅 6 增 14 删（注释 + 删除裁剪）；`controller` 端点用例断言字段（`check_enabled`/`apply_enabled`/`state`/`latest_version`/`has_update`/`rollback_available`）全部通过 | 通过 |
| 非测试代码无人依赖被裁列表 | 全仓库非测试代码中 `.Assets` 仅出现在 `service/update_apply.go:200/205/210`；`ReleaseInfo` 只作为 fetcher 返回与 checker 内存缓存，从不序列化给客户端 | 通过 |
| 前后端码字面量一致 | 后端 `UpdateApplyCodeSignatureAssetMissing = "signature_asset_missing"`（`service/update_apply.go:31-33`）；前端 `FAILURE_CODES['signature_asset_missing'] = 'signature_asset_missing'`（`update-api.ts:128`）与联合类型成员（`:79`）；控制器把 `applyErr.Code` 原样写入响应 `code` 字段（`controller/update_check.go:84`），大小写完全一致 | 通过 |
| 缺 sig 仍 fail-closed | `service/update_apply.go:210-219`：缺失即 `return`（502），其后再无放行分支；验签（`:238`）发生在下载二进制之前，验签不过不创建临时文件；未签名/无公钥一律拒绝 | 通过 |
| T2–T5 仍为 `signature_mismatch` | T2 `TestApplyUpdateRejectsMalformedSignature`、T3 `TestApplyUpdateRejectsForgedSignature`、T4 `TestApplyUpdateRejectsSignatureOverDifferentChecksums` 断言 `signature_mismatch`；T5 `TestVerifySHA256SumsSignatureRejectsEmptyKeyList` 断言空公钥列表下验签必须报错 | 通过 |
| 日志含缺失资产名、不含敏感字节 | 代码只记录编译期常量 `UpdateAssetNameSHA256SumsSig`（`service/update_apply.go:213-216`）；`common.SysError` 写 `gin.DefaultErrorWriter` 且自身不脱敏（`common/sys_log.go:24-29`）；新用例断言日志含资产名、且不含 `public key`、`signature does not match`、内嵌公钥串 | 通过 |

## 四、安全与脱敏取舍（独立复核 Lead 提供的事实）

在 /tmp 克隆里加临时探针实测（未改真实工作区）：

| 输入 | 输出 |
|---|---|
| `SHA256SUMS.sig` | `***.sig` |
| `SHA256SUMS` | `SHA256SUMS` |
| `update apply signature asset is missing: SHA256SUMS.sig` | `update apply signature asset is missing: ***.sig` |

结论：Lead 提供的事实成立。若对该日志套 `MaskSensitiveInfo`，唯一有诊断价值的信息（缺的是哪个资产）会被抹掉，
因此「不脱敏」的取舍可以接受。风险面也已被限定：写入日志的是编译期常量、发布侧公开文件名，
不含凭据、公钥字节或签名字节；同文件其它日志（源不可用、验签失败、回滚点失败）仍保留脱敏，未放宽。

## 五、七语言核对

| 核对项 | 结果 |
|---|---|
| 每文件 diff 行数 | en / zh / zh-TW / fr / ja / ru / vi 均为 `1 insertion, 0 deletions`（纯插入，无任何已有行被移动） |
| 键存在性 | 组件里的 t() 串与 7 个 locale 的键**逐字符相同**；7 个文件均有该键，无空值译文本 |
| 插入位置 | 7 个文件均在索引 4235，前后邻居同为 `System task records` / `TPM`，且 `localeCompare` 判定前后序均正确（项目 sync 脚本用同一比较器） |
| JSON 合法性 | 7 个文件 `jq -e` 全部 OK；各文件总键数 5318 → 5319（+1） |
| i18n 门禁 | `bun run i18n:check` 通过 |

## 六、边界自查

- `git status --porcelain`：恰好 15 个已跟踪改动 + 1 个未跟踪任务目录
  `.trellis/tasks/10-24-fix-update-asset-whitelist/`；**除任务目录外无任何未跟踪残留**（含 `-uall` 复核）。
- 真实工作区未被本次验证修改：diff 总量与基线一致（193 增 / 20 删，`diff sha256` 见头部）；
  所有破坏性操作只在 `/tmp/verify-repo` 克隆内进行。
- 测试自身不触网：新用例用 `httptest` 回环 server，并显式把 transport 代理置空、`DialContext` 只放行回环地址
  （非回环直接报错拒绝），我已读该守卫实现确认。

## 七、非阻断观察与不确定项

1. （观察）`controller/update_check_endpoint_test.go:58-88` 的稳定 code 枚举用例未把新码
   `UpdateApplyCodeSignatureAssetMissing` 加入清单；功能无缺口（新码的 502 由 service 层用例断言），
   但把新码补进该表可让「响应体带稳定 code」这条契约覆盖到它。
2. （观察）`filterUpdateAssets` 已删除且无残留，PRD 的 R2 表述是「裁剪移到状态响应组装处」，
   实现改为**完全不裁剪**：因 `UpdateCheckStatus` 从不含 assets，效果等价且更简单；已核无其它消费方依赖被裁列表。
3. （观察）fetcher 现在把完整资产列表缓存进内存 `ReleaseInfo`（此前只留 3 项）。量级为单次 Release 的资产数组，
   无响应体扩大（结构体无 assets 字段）、无内存风险；如未来把 `ReleaseInfo` 直接序列化，需重新评估。
4. （不确定项）本轮**无法对真实 GitHub Release 端点验证**（红线禁网）。直通用例以 httptest 模拟 GitHub 响应，
   覆盖「fetcher 解析 → applier 取 sig → 进验签」的完整链路，但不覆盖 GitHub 真实响应字段差异。
5. （不确定项）受影响实例（v0.0.43–45 体内带此 bug）无法靠面板自更新到修复版，需先以镜像方式升级一次；
   属发布/运维动作，不在本改动验证范围（PRD Notes 已记录）。

## 八、复跑命令清单

```bash
# 修复前后对照（Go）
rm -rf /tmp/verify-repo && git clone -q /root/new-api-own /tmp/verify-repo
cp /root/new-api-own/service/update_apply_test.go /root/new-api-own/service/update_check_test.go /tmp/verify-repo/service/
# 仅在 /tmp 克隆内加「只定义新常量」的 shim 以取得行为对照（不触碰真实工作区）
printf 'package service\n\nconst UpdateApplyCodeSignatureAssetMissing = "signature_asset_missing"\n' > /tmp/verify-repo/service/zz_prefix_shim_test.go
cd /tmp/verify-repo && GOPROXY=off go test ./service/ -run 'SignatureAsset|Signature|Fetcher' -count=1 -v

# 真实工作区门禁
cd /root/new-api-own && GOPROXY=off go build ./... && GOPROXY=off go vet ./... && GOPROXY=off go test ./service/... -count=1
go test ./service/ -run 'SignatureAsset|Signature|Fetcher' -count=1 -v
go test ./controller/ -run 'Update' -count=1
cd web && bun run typecheck && bun run i18n:check && bun test src/features/system-settings/maintenance/__tests__/
```
