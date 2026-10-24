# 后端证据：自更新资产裁剪与错误码修复（A+B+C1+C3）

- 任务：Trellis `10-24-fix-update-asset-whitelist`；共享任务板 `task-19`
- 执行者：worker-script；时间：2026-10-24
- 写作用域：`service/update_check.go`、`service/update_apply.go`、`service/update_check_test.go`、
  `service/update_apply_test.go` + 本文件。**未**改 `controller/`、`setting/`、`web/`；未提交、未推送、未发版。
- 网络：新增测试只用 `httptest` 回环（并在自定义 `DialContext` 里显式拒绝非回环地址），**没有任何外部主机访问**。

## 1. 根因链条（修复前后）

| 环节 | 修复前 | 修复后 |
| --- | --- | --- |
| `githubReleaseFetcher.FetchLatestRelease`（`service/update_check.go:276` 原） | 调 `filterUpdateAssets` 把资产裁到 3 项（白名单只有 `SHA256SUMS` + amd64/arm64） | **不裁剪**，原样返回完整资产列表 |
| `filterUpdateAssets`（原 `:289-299`） | 白名单漏 `SHA256SUMS.sig` | 函数删除（唯一用途就是这个 bug 来源） |
| `UpdateApplier.Apply` 资产检查（`service/update_apply.go:207`） | `findUpdateAsset(..., "SHA256SUMS.sig")` 恒 not-found → 稳定 `signature_mismatch`(502) | 拿到 `.sig`，进入验签阶段 |
| 缺 `.sig` 的报码 | `signature_mismatch`（与验签失败混在一起，误导读日志的人） | 独立 `signature_asset_missing`(502) + `SysError` 记缺失资产名 |

## 2. 改动清单

1. **`service/update_check.go`**
   - 删除 `release.Assets = filterUpdateAssets(release.Assets)`（fetcher 不再裁剪，改为注释说明为何**不能**裁剪：同一 fetcher 也是 applier 的数据源）。
   - 删除 `filterUpdateAssets` 函数本体。
   - `ReleaseInfo` 增补注释：`Assets` 保持完整列表。
2. **`service/update_apply.go`**
   - 新增 `UpdateApplyCodeSignatureAssetMissing = "signature_asset_missing"`。
   - 缺 `.sig` 分支改用新码（HTTP 502 不变），并 `common.SysError("update apply signature asset is missing: SHA256SUMS.sig")`。
   - faile-closed 语义不变：仍拒绝替换、仍不降级放行。
3. **`service/update_check_test.go`**
   - `stableReleasePayload` 增补 `SHA256SUMS.sig` 资产。
   - `TestGithubReleaseFetcherParsesStableRelease` 改为断言 **fetcher 返回完整资产列表**（含 `.sig`、`notes.txt`）。修复前该断言正是把 bug 锁死的那处（原断言 `Len(Assets)==2`）。
4. **`service/update_apply_test.go`**
   - 新增 `TestApplyUpdateSeesSignatureAssetFromRealFetcher`：**fetcher → applier 直通回归**（httptest 假 Release + 真实 fetcher + 真实 HTTP 下载器包装的记录器）。
   - 新增 `TestApplyUpdateMissingSignatureAssetReportsCodeAndLogsAssetName`：新错误码 + 502 + 日志含资产名 + 不含公钥/验签细节。
   - `TestApplyUpdateRejectsMissingSignatureAsset` 期望码改为 `signature_asset_missing`。
   - 新增 `recordingHTTPDoer`、`serverDialContext`（仅回环）测试基础设施。

关键 diff 摘要（production）：

```diff
--- a/service/update_check.go
-	release.Assets = filterUpdateAssets(release.Assets)
+	// 注意：这里**不裁剪资产**。…（历史缺陷：白名单漏掉 SHA256SUMS.sig → 自更新 100% 失败）
 	return release, nil

--- a/service/update_apply.go
+	UpdateApplyCodeSignatureAssetMissing = "signature_asset_missing"
 	signatureAsset, ok := findUpdateAsset(release.Assets, UpdateAssetNameSHA256SumsSig)
 	if !ok || strings.TrimSpace(signatureAsset.DownloadURL) == "" {
+		common.SysError(fmt.Sprintf("update apply signature asset is missing: %s", UpdateAssetNameSHA256SumsSig))
-		return nil, newUpdateApplyError(UpdateApplyCodeSignatureMismatch, http.StatusBadGateway,
+		return nil, newUpdateApplyError(UpdateApplyCodeSignatureAssetMissing, http.StatusBadGateway,
```

## 3. 修复前后对照证据（C3 要求）

方法：保持**新测试不变**，把 `service/update_check.go` / `service/update_apply.go` 临时还原为
`git` 原版（`git checkout --`，即剔掉 A+B+C1），跑同一组用例；再把修复版本恢复（sha256 校验一致）后重跑。
新常量在未修复代码上不存在，故对照跑时临时把测试里的 `UpdateApplyCodeSignatureAssetMissing`
换成字面量 `"signature_asset_missing"`，跑完立即换回（当前文件里是常量引用）。

修复前（生产代码 = v0.0.45 状态）：

```console
--- PASS: TestApplyUpdateRejectsForgedSignature (0.00s)
--- PASS: TestApplyUpdateRejectsSignatureOverDifferentChecksums (0.00s)
--- FAIL: TestApplyUpdateRejectsMissingSignatureAsset (0.00s)
--- FAIL: TestApplyUpdateSeesSignatureAssetFromRealFetcher (0.00s)
--- FAIL: TestGithubReleaseFetcherParsesStableRelease (0.00s)
FAIL	github.com/QuantumNous/new-api/service	0.027s
```

三条失败的具体断言（证明缺陷被真正复现，而不是"改测试让它过"）：

```console
# 1) 直通回归：applier 一个下载请求都没发出（.sig 在 fetcher 里已被裁掉）
    update_apply_test.go:542:
        Error:  []string(nil) does not contain "http://127.0.0.1:39683/assets/SHA256SUMS.sig"
        Messages: applier 必须拿到并请求签名资产；没有该请求说明 fetcher 仍在裁剪资产

# 2) 缺 sig 时错误码仍是 signature_mismatch（新码不存在）
    expected: "signature_asset_missing"
    actual  : "signature_mismatch"

# 3) fetcher 返回的资产被裁到 2 项
    expected: []string{"SHA256SUMS", "SHA256SUMS.sig", "new-api-linux-amd64", "notes.txt"}
    actual  : []string{"SHA256SUMS", "new-api-linux-amd64"}
```

修复后（同一组用例，逐条）：

```console
--- PASS: TestApplyUpdateRejectsForgedSignature (0.00s)
--- PASS: TestApplyUpdateRejectsSignatureOverDifferentChecksums (0.00s)
--- PASS: TestApplyUpdateRejectsMissingSignatureAsset (0.00s)
--- PASS: TestApplyUpdateMissingSignatureAssetReportsCodeAndLogsAssetName (0.00s)
--- PASS: TestApplyUpdateSeesSignatureAssetFromRealFetcher (0.00s)
--- PASS: TestApplyUpdateRejectsMalformedSignature (0.00s)
--- PASS: TestVerifySHA256SumsSignatureRejectsEmptyKeyList (0.00s)
--- PASS: TestGithubReleaseFetcherParsesStableRelease (0.00s)
ok  	github.com/QuantumNous/new-api/service	0.030s
```

**直通回归为什么能证明修复**：该用例把真实 fetcher（httptest 上的 GitHub 风格 Release）与真实
applier 串起来，用 `recordingHTTPDoer` 记录 applier 实际请求过的 URL。修复前 applier 连
`.sig` 都不请求（`[]string(nil)`）；修复后它请求了 `.sig` 并停在验签失败（签名故意覆盖另一份内容）。
即「越过资产检查、进入验签阶段」，与 design.md §3.2 的判据一致。

## 4. 门禁输出（全部实际执行）

```console
$ go build ./...
（无输出，exit 0）

$ go vet ./...
（无输出，exit 0）

$ go test ./service/... -count=1
ok  	github.com/QuantumNous/new-api/service	0.456s
ok  	github.com/QuantumNous/new-api/service/authz	0.024s
?   	github.com/QuantumNous/new-api/service/passkey	[no test files]

$ go test ./... -count=1        # 全仓回归，确认没有其它包被打断
（除 ok/? 外无输出 = 全绿）
```

既有 fail-closed 用例保持全绿（未降级放行）：`TestApplyUpdateRejectsForgedSignature`、
`TestApplyUpdateRejectsSignatureOverDifferentChecksums`、`TestApplyUpdateRejectsMalformedSignature`、
`TestVerifySHA256SumsSignatureRejectsEmptyKeyList`、`TestApplyUpdateChecksumMismatchKeepsBinary` 等
均在 `go test ./service/...` 中通过。

## 4b. 稳定 code 清单补全（verifier 复核后的非阻断小补）

复核发现：`controller/update_check_endpoint_test.go` 的 `TestUpdateErrorResponsesCarryStableCode`
是「所有对外稳定 code」的清单，已含 `service.UpdateApplyCodeSignatureMismatch`，
但新引入的 `service.UpdateApplyCodeSignatureAssetMissing` 漏登记。已在 `:65` 行后补一行
（与既有写法一致，未改动其它任何 case 或断言逻辑）：

```diff
 		{code: service.UpdateApplyCodeSignatureMismatch, status: http.StatusBadGateway},
+		{code: service.UpdateApplyCodeSignatureAssetMissing, status: http.StatusBadGateway},
 		{code: service.UpdateApplyCodeBackupFailed, status: http.StatusInternalServerError},
```

验证（新子用例已出现在 `-v` 输出中，且整组通过）：

```console
$ go test ./controller/ -run TestUpdateErrorResponsesCarryStableCode -count=1 -v
=== RUN   TestUpdateErrorResponsesCarryStableCode
=== RUN   TestUpdateErrorResponsesCarryStableCode/signature_mismatch
=== RUN   TestUpdateErrorResponsesCarryStableCode/signature_asset_missing
=== RUN   TestUpdateErrorResponsesCarryStableCode/backup_failed
=== RUN   TestUpdateErrorResponsesCarryStableCode/no_rollback_point
=== RUN   TestUpdateErrorResponsesCarryStableCode/rollback_point_invalid
=== RUN   TestUpdateErrorResponsesCarryStableCode/rollback_failed
=== RUN   TestUpdateErrorResponsesCarryStableCode/disabled
=== RUN   TestUpdateErrorResponsesCarryStableCode/not_newer
=== RUN   TestUpdateErrorResponsesCarryStableCode/checksum_mismatch
--- PASS: TestUpdateErrorResponsesCarryStableCode (0.00s)
ok  	github.com/QuantumNous/new-api/controller	0.012s

$ go vet ./controller/
（无输出，exit 0）
```

该补丁把新码纳入「对外契约」回归网：若今后有人改动 `writeUpdateApplyError` 使新码丢掉
`code` 字段或改错状态码，这个用例会直接失败。写作用域说明：`controller/update_check_endpoint_test.go`
本就在 task-19 的 advisory 写作用域内（"必要时 controller/update_check.go" 的同层测试），
改动仅此一行，未顺带动其它内容。

## 5. 两处必须报给 Lead 的偏差点

### 偏差 1（B）：代码里**不存在**「状态响应组装处」

任务与 design.md 假设"裁剪移出 fetcher、只在状态响应组装处裁剪"。实际核查结论：

- `UpdateCheckStatus`（`service/update_check.go:90-102`）**没有任何资产字段**，
  `controller/update_check.go:24` 直接 `common.ApiSuccess(c, service.GetUpdateStatus(...))`。
- 全仓 `Assets` 的消费点只有一处：`service/update_apply.go:197/202/207`（applier）。
- 因此"在响应组装处裁剪"没有落点，且**不必要**：对外响应本来就不含资产，裁剪本身是纯内部行为。

采取的等价做法（满足 B 的意图）：**fetcher 返回全量**（完整性回到数据源），
`filterUpdateAssets` 直接删除（它的唯一用途就是这个 bug），响应结构一行未动。
响应形状风险的兜底证据：`controller/update_check_endpoint_test.go` 全绿（断言
`state`/`latest_version`/`has_update`/`rollback_available` 等字段形状），且 `UpdateCheckStatus`
的字段列表本次未被修改（`git diff` 里无该 struct 变化）。

### 偏差 2（C1）：日志**未**用 `MaskSensitiveInfo` 包装资产名

任务写"动态信息用 `common.MaskSensitiveInfo` 包"，但实测该函数把资产名当作域名打码：

```console
SHA256SUMS.sig    -> "***.sig"
SHA256SUMS        -> "SHA256SUMS"
new-api-linux-amd64 -> "new-api-linux-amd64"
```

即包装后会丢掉"缺的是哪个资产"这一信息，恰好抹掉独立错误码存在的**唯一诊断价值**
（本次误诊就是被 `signature_mismatch` 的模糊文案拖住的）。

处置：日志打**未脱敏的资产名**。理由与边界：资产名是编译期常量、发布侧的公开文件名，
不含凭据、签名字节或公钥；其余动态串（错误原因、URL）仍照旧走 `MaskSensitiveInfo`。
测试显式断言日志**不含**内嵌公钥、`public key`、`signature does not match`。
若 Lead 更偏向"一律套 MaskSensitiveInfo"，一行即可切回（日志将变成 `***.sig`）。

## 6. 风险与未完成

- **未做真实实例/真实 Release 验证**：按红线未访问外部主机；httptest 只覆盖回环。
  生产链路的最终确认需在授权后按发布流程演练（属 P1/P2 授权范围）。
- **未提交**：改动留在工作区（`git status` 里 `service/*.go` 四个文件）。
- 新增测试基础设施 `serverDialContext` 在非回环地址上直接报错，测试不可能意外外呼。
- 前端错误码映射（C2）由 worker-docs 负责：本端已保证新码字面量为 `signature_asset_missing`、
  HTTP 502、`code` 字段沿用 `writeUpdateApplyError` 的既有形状（`success/message/code`），前端零结构改动。
- 注意：`web/scripts/add-missing-keys.mjs` 是**未跟踪的新文件**（worker-docs 侧产物），不属本任务写作用域，
  提交前请确认是否需要保留。
- 本次曾发生一次自伤：我在做修复前后对照时用 `git checkout --` 还原生产文件，误把 `/tmp` 里的**原版**
  备份当成"我的修复版"恢复，导致修复一度被撤。已重新应用并以 sha256 校验复原
  （`update_check.go 601ebf65…`、`update_apply.go 034db6ba…`，与复核过的版本一致），门禁已重跑全绿。
  记录在此以免 Lead 复核 diff 时困惑。

## 7. 建议下一步

1. Lead/verifier 复核本报告 + `git diff -- service/`，重点看第 5 节两处偏差是否接受。
2. 与 worker-docs 对齐 `signature_asset_missing` 字面量与 i18n key（前端映射必须与后端码一致）。
3. 门禁与前端检查通过后由 Lead 提交（锚点 `[task:fix-update-asset-whitelist]`）。
4. 若要防同类回归：可考虑给 `ReleaseFetcher` 接口加注释契约"必须返回完整资产列表"，
   或在 CI 里显式保留本直通用例（已随代码入库，无需额外配置）。
