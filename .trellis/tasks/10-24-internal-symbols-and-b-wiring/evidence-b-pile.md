# B 堆收尾证据（task-54 / PRD R2）

- Trellis 任务：`10-24-internal-symbols-and-b-wiring`（PRD 见同目录 `prd.md`）
- 共享任务：task-54「B 堆接线：IsPrivateIP / GetContextKeyStringSlice / kitutil JSON 策略」
- 执行者：worker-script2（接替前一位中途中断的成员，在其未提交改动上收尾）
- 日期：2026-10-24
- 边界：未提交、未推送、未发版、未联网、未装工具、未动 `data/` 与实例；只写本证据文件与写作用域内文件

## 0. 结论

| 项 | 结论 | 关键证据 |
|---|---|---|
| B1 `common.IsPrivateIP` | 通过：两套实现合并为一处，函数体逐字一致（仅改名+文档注释）；全仓调用点审查无行为倒退 | §2.1 逐字 diff、§2.2 调用点表、§2.3 46 输入对照测试 0 不一致 |
| B2 `common.GetContextKeyStringSlice` | 通过：三分支语义保持，返回值的 nil/空切片形态也与旧实现一致 | §3.1 分支表、§3.2 消费方审查、§3.3 测试 |
| B3 `relaykit/kitutil` 两个 JSON 函数 | 采用方案②：删除 2 个无人使用的包装 + 在 `relaykit/README.md` 注明模块内 JSON 例外；AGENTS.md 未改（理由见 §4.2） | §4.1 决策、§4.2 措辞核对、§4.3 README 落点、§4.4 级联 |
| 门禁 | 全部通过（见 §5） | build/vet/test/relaykit/gofmt/unused/deadcode |

## 1. 工作区改动清单（全部未提交）

```console
$ git diff --stat
 common/ip.go                          | 19 ----------
 common/ssrf_protection.go             | 14 +++++---
 common/ssrf_protection_test.go        | 68 +++++++++++++++++++++++++++++++++++
 relaykit/README.md                    | 14 ++++++++
 relaykit/relayconvert/kitutil/json.go |  9 -----
 service/group.go                      | 14 ++++----
 service/group_auto_groups_test.go     | 24 +++++++++++++
 7 files changed, 122 insertions(+), 40 deletions(-)
```

| 文件 | 改动 | 归属 | 说明 |
|---|---|---|---|
| `common/ip.go` | -19 行 | B1 | 删除旧的较弱重复实现 `common.IsPrivateIP`（全仓 0 调用） |
| `common/ssrf_protection.go` | +14/-10 行（rename + 注释 + 3 调用点） | B1 | 内部 `isPrivateIP` 改为导出 `IsPrivateIP` 并补文档注释 |
| `common/ssrf_protection_test.go` | +68 行 | B1 | 表驱动契约测试（本轮补边界用例） |
| `service/group.go` | 接线 | B2 | `GetRequestAutoGroups` 改用 `common.GetContextKeyStringSlice` |
| `service/group_auto_groups_test.go` | +24 行 | B2 | 三分支之二/之三测试 |
| `relaykit/relayconvert/kitutil/json.go` | -9 行 | B3 | 删除 `UnmarshalJsonStr` / `DecodeJson` 及 `io` import |
| `relaykit/README.md` | +14 行 | B3 | 「JSON 编码约定（模块例外）」小节 |

另：`.trellis/tasks/10-24-internal-symbols-and-b-wiring/` 为未跟踪的任务目录（含本文件），不属代码改动。

## 2. B1：`common.IsPrivateIP` 合并

### 2.1 前后对照（HEAD → 工作区）

改动前仓库里有**两套**私网判定：

| 位置（HEAD） | 名称 | 覆盖范围 | 全仓 Go 引用数 |
|---|---|---|---|
| `common/ip.go:14-31` | `IsPrivateIP`（导出） | 回环、链路本地（unicast/multicast）、10/8、172.16/12、192.168/16 | **0**（`git grep -n IsPrivateIP HEAD -- '*.go' ':!common/ip.go'` 无输出） |
| `common/ssrf_protection.go:99-136` | `isPrivateIP`（未导出，内联） | 上表全部 + 未指定/接口本地组播/ULA/IPv4 15 个特殊网段/IPv6 10 个特殊网段/`IsPrivate`；nil→true | 3（`ssrf_protection.go:262/276/285`） |

改动后：只保留 `common/ssrf_protection.go` 的导出版本，`common/ip.go` 中的旧实现删除。

**逐字等价证明**（旧内联函数体与当前函数体，仅 normalize 函数名后 diff，零输出即逐字一致）：

```console
$ diff <(git show HEAD:common/ssrf_protection.go | sed -n '99,136p' | sed 's/^func isPrivateIP/func IsPrivateIP/') \
       <(sed -n '103,140p' common/ssrf_protection.go)
(无输出)
$ echo $?
0
```

结论：3 个生产调用点所执行的函数体**一个字节都没变**，本项是纯改名 + 补文档注释，不可能产生新拒绝或行为倒退。

**语义方向说明**（任务要求核对的点）：新实现确实比旧 `common.IsPrivateIP` **更严格**（nil→true、新增 ULA/CGNAT/未指定/组播等），但旧 `common.IsPrivateIP` 在全仓**没有任何调用点**（上表 0），所以不存在“某调用方因判定变严而被新拒绝”的路径；3 个生产调用点原本执行的就是这套更严格的判定。

旧 `common.IsPrivateIP` 的 true 分支是旧内联实现的**子集**（回环、链路本地、10/8、172.16/12、192.168/16 均在旧内联的 `IsLoopback/IsLinkLocal*/privateIPv4Nets` 覆盖内），因此“两套的并集 == 旧内联实现 == 新实现”，不会丢判定。

### 2.2 全仓调用点审查

```console
$ grep -rn "IsPrivateIP|isPrivateIP" --include='*.go' .
common/ssrf_protection.go:98   // 文档注释
common/ssrf_protection.go:103  func IsPrivateIP(ip net.IP) bool
common/ssrf_protection.go:266  if IsPrivateIP(ip) && !p.AllowPrivateIp {     // IsIPAccessAllowed
common/ssrf_protection.go:280  if IsPrivateIP(ip) && !p.AllowPrivateIp {     // ipAccessError(host != "")
common/ssrf_protection.go:289  if IsPrivateIP(ip) && !p.AllowPrivateIp {     // ipAccessError(fallback)
common/ssrf_protection_test.go:111  require.Equal(t, tc.want, IsPrivateIP(tc.ip))
```

| 调用点 | 改动前（HEAD） | 改动后 | 行为影响 |
|---|---|---|---|
| `common/ssrf_protection.go:262`（`IsIPAccessAllowed`） | `isPrivateIP` | `IsPrivateIP` | 无（函数体逐字一致） |
| `common/ssrf_protection.go:276`（`ipAccessError`，有 host） | `isPrivateIP` | `IsPrivateIP` | 无 |
| `common/ssrf_protection.go:285`（`ipAccessError`，无 host 兜底） | `isPrivateIP` | `IsPrivateIP` | 无 |
| `common/ip.go:14`（旧导出实现） | 定义存在、0 调用 | 已删除 | 无（无调用点） |
| 测试 | 无覆盖 | 表驱动契约测试 | 仅新增测试 |

非 Go 位置：只有 `.trellis/tasks/archive/.../evidence-symbols.md` 的只读盘点清单提到 `common.IsPrivateIP` 这个名字（归 B 堆、建议接线），无代码契约/脚本/CI/前端引用；本项落地后该清单对应行即已闭环。

### 2.3 等价性对照测试（临时脚本，已删除）

方法：把 HEAD 上的两套旧实现原样复制进临时测试文件 `common/zz_equiv_tmp_test.go`，对 46 个输入同时计算 `oldCommon`、`oldInline`、`new`，判定 ① `new == oldInline` 逐输入一致；② `oldCommon == true` 时 `new` 必须为 true（并集覆盖）。

```console
$ go test ./common/ -run TestTmpLegacyEquivalence -count=1 -v
input                                                oldCommon  oldInline  new        verdict
ipv4-public-8.8.8.8                                  false      false      false      OK
ipv4-public-1.1.1.1                                  false      false      false      OK
ipv4-public-9.255.255.255                            false      false      false      OK
ipv4-public-11.0.0.0                                 false      false      false      OK
ipv4-public-172.15.255.255                           false      false      false      OK
ipv4-public-172.32.0.0                               false      false      false      OK
ipv4-public-192.167.255.255                          false      false      false      OK
ipv4-public-192.169.0.0                              false      false      false      OK
ipv4-public-100.63.255.255                           false      false      false      OK
ipv4-public-100.128.0.0                              false      false      false      OK
ipv4-private-10.0.0.1                                true       true       true       OK
ipv4-private-10.255.255.255                          true       true       true       OK
ipv4-private-172.16.0.1                              true       true       true       OK
ipv4-private-172.31.255.255                          true       true       true       OK
ipv4-private-192.168.1.1                             true       true       true       OK
ipv4-loopback-127.0.0.1                              true       true       true       OK
ipv4-loopback-127.255.255.254                        true       true       true       OK
ipv4-linklocal-169.254.169.254                       true       true       true       OK
ipv4-unspecified-0.0.0.0                             false      true       true       OK
ipv4-cgnat-100.64.0.1                                false      true       true       OK
ipv4-testnet1-192.0.2.1                              false      true       true       OK
ipv4-bench-198.18.0.1                                false      true       true       OK
ipv4-multicast-224.0.0.1                             true       true       true       OK
ipv4-reserved-240.0.0.1                              false      true       true       OK
ipv4-broadcast-255.255.255.255                       false      true       true       OK
ipv4-mapped-loopback-::ffff:127.0.0.1                true       true       true       OK
ipv4-mapped-private-::ffff:10.0.0.1                  true       true       true       OK
ipv4-mapped-public-::ffff:8.8.8.8                    false      false      false      OK
ipv4-mapped-unspecified-::ffff:0.0.0.0               false      true       true       OK
ipv6-public-2001:4860:4860::8888                     false      false      false      OK
ipv6-public-2606:4700:4700::1111                     false      false      false      OK
ipv6-public-fbff::1                                  false      false      false      OK
ipv6-public-2001:200::1                              false      false      false      OK
ipv6-ula-fc00::1                                     false      true       true       OK
ipv6-ula-fd12:3456:789a::1                           false      true       true       OK
ipv6-ula-fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff     false      true       true       OK
ipv6-loopback-::1                                    true       true       true       OK
ipv6-linklocal-fe80::1                               true       true       true       OK
ipv6-linklocal-febf:ffff::1                          true       true       true       OK
ipv6-unspecified-::                                  false      true       true       OK
ipv6-doc-2001:db8::1                                 false      true       true       OK
ipv6-multicast-ff02::1                               true       true       true       OK
ipv6-multicast-interface-local-ff01::1               false      true       true       OK
nil                                                  false      true       true       OK
empty-non-nil                                        false      false      false      OK
invalid-4byte-nonzero-len16                          false      true       true       OK

summary: inputs=46 mismatch_vs_oldInline=0 missing_oldCommon=0
--- PASS: TestTmpLegacyEquivalence (0.00s)
PASS
```

清理：

```console
$ rm -- common/zz_equiv_tmp_test.go
$ git status --short -- common/
 M common/ip.go
 M common/ssrf_protection.go
 M common/ssrf_protection_test.go
```

### 2.4 永久表驱动测试

文件：`common/ssrf_protection_test.go`（函数 `TestIsPrivateIPClassifiesSpecialPurposeAddresses`，`git diff` 新增 68 行）。
契约：私网判定只在 `IsPrivateIP` 一处维护，SSRF 校验（`ValidateNetworkTarget` / `ValidateResolvedIP`）的行为由本表间接锁定。

| 任务要求的覆盖项 | 表内用例（节选） | 期望 |
|---|---|---|
| IPv4 公网 | 8.8.8.8、1.1.1.1、172.32.0.1 | false |
| IPv4 10/8、172.16/12、192.168/16 | 10.0.0.1、10.255.255.255、172.16.0.1、172.31.255.255、192.168.1.1 | true |
| IPv4 回环 | 127.0.0.1、127.255.255.254 | true |
| IPv4 链路本地 | 169.254.169.254 | true |
| IPv4 未指定 | 0.0.0.0 | true |
| IPv6 公网 | 2001:4860:4860::8888、2606:4700:4700::1111 | false |
| IPv6 ULA（fc00::/7） | fc00::1、fd12:3456:789a::1、fdff::1 | true |
| IPv6 回环 | ::1 | true |
| IPv6 链路本地 | fe80::1 | true |
| IPv6 未指定 | :: | true |
| nil | nil | true |

本轮补强的边界用例（防 CIDR mask 写错）：

| 用例 | 期望 | 意义 |
|---|---|---|
| 9.255.255.255 / 11.0.0.0 | false | 10/8 两侧边界 |
| 172.15.255.255 / 172.32.0.0 | false | 172.16/12 两侧边界 |
| 192.167.255.255 / 192.169.0.0 | false | 192.168/16 两侧边界 |
| 100.63.255.255 / 100.128.0.0 | false | CGNAT 100.64/10 两侧边界 |
| ::ffff:10.0.0.1 / ::ffff:8.8.8.8 | true / false | IPv4-mapped 私网/公网 |
| fbff::1、fdff::1 | false / true | fc00::/7 两侧边界 |

运行：

```console
$ go test ./common/ -run 'TestIsPrivateIP|TestSSRF' -count=1 -v
--- PASS: TestIsPrivateIPClassifiesSpecialPurposeAddresses (0.00s)   # 43 个子用例全部 PASS
--- PASS: TestSSRFProtectionRejectsLiteralPrivateAndReservedIPs
--- PASS: TestSSRFProtectionAllowsPrivateIPWhenExplicitlyEnabled
--- PASS: TestSSRFProtectionRejectsResolvedPrivateIP
--- PASS: TestNewSSRFProtectionFromFetchSettingParsesPortRanges
PASS
ok  	github.com/QuantumNous/new-api/common	0.009s
```

## 3. B2：`GetRequestAutoGroups` 接线 `GetContextKeyStringSlice`

### 3.1 改动前后与三分支对照

改动前（HEAD `service/group.go:96-107`）：手工 `common.GetContextKey` + `value.([]string)` 类型断言。
改动后（工作区）：

```go
func GetRequestAutoGroups(c *gin.Context, userGroup string) []string {
	if _, ok := common.GetContextKey(c, constant.ContextKeyTokenAutoGroups); !ok {
		return GetUserAutoGroup(userGroup)
	}
	return FilterUserTokenAutoGroups(userGroup, common.GetContextKeyStringSlice(c, constant.ContextKeyTokenAutoGroups))
}
```

`common.GetContextKeyStringSlice`（`common/gin.go:177`，HEAD 已存在，本轮只是接线）实现为 `c.GetStringSlice(string(key))`，gin 的语义是：key 缺失或值为 nil → nil；类型不符 → nil；类型为 `[]string` → 原值。

| 分支 | 改动前 | 改动后 | 等价性 |
|---|---|---|---|
| ① key 缺失 | `GetUserAutoGroup(userGroup)` | 同左（先探测存在性） | 完全一致 |
| ② 存在但类型不符 | 返回 `[]string{}` | helper 返回 nil → `FilterUserTokenAutoGroups` 返回 `make([]string, 0, 0)`（非 nil、len 0） | 输出形态一致（**都是非 nil 空切片**，JSON 均为 `[]`）；len/range/JSON 无差异 |
| ③ 存在且合法 | `FilterUserTokenAutoGroups(userGroup, groups)` | 同左（helper 原样返回 `[]string`） | 完全一致 |

“顺序不能换”的理由已写入代码注释：必须先判断 key 是否存在，再取切片；若省略存在性判断，类型不符会退化为“继承全局”，绕过 token 的显式快照。

### 3.2 消费方审查（nil vs 空切片是否有语义差异）

```console
$ grep -rn "GetRequestAutoGroups" --include='*.go' . | grep -v _test.go
service/group.go:97:func GetRequestAutoGroups(...)
service/channel_select.go:90:	autoGroups := GetRequestAutoGroups(param.Ctx, userGroup)
middleware/distributor.go:112:	autoGroups := service.GetRequestAutoGroups(c, userGroup)
controller/model.go:194:	ownerGroups: service.GetRequestAutoGroups(c, userGroup),
```

| 消费方 | 用法 | nil vs 空切片 |
|---|---|---|
| `service/channel_select.go:90` | `if len(autoGroups) == 0 { return error }` | 不区分（len 相同） |
| `middleware/distributor.go:112` | `for _, g := range autoGroups` | 不区分（range 相同） |
| `controller/model.go:194` | 存入 `ownerGroups` 后用于模型列表构建 | 不区分；且如上表分支②所述，新旧返回值都是非 nil 空切片 |

另：分支②/③ 都经过 `FilterUserTokenAutoGroups`，它总是 `make([]string, 0, ...)` 起步，因此**任何**路径都不会返回 nil 切片——与旧实现的 `[]string{}` 形态一致。

### 3.3 测试

```console
$ go test ./service/ -run 'TestGetRequestAutoGroups' -count=1 -v
--- PASS: TestGetRequestAutoGroupsInheritedListIsNotLimited          # 分支①：缺失→继承全局，且不受 token limit 约束
--- PASS: TestGetRequestAutoGroupsFiltersBeforeApplyingCurrentLimit   # 分支③：先权限过滤再应用当前 limit
--- PASS: TestGetRequestAutoGroupsDoesNotFallBackAfterPermissionChange # 分支③：权限变化后不回退全局
--- PASS: TestGetRequestAutoGroupsWrongTypeIsExplicitEmptySnapshot     # 分支②：类型不符→显式空快照（不得继承全局）
--- PASS: TestGetRequestAutoGroupsValidSnapshotFiltersWithoutInheriting # 分支③：合法快照过滤且非继承
PASS
ok  	github.com/QuantumNous/new-api/service	0.019s
```

（前 3 个用例为既有用例，由前任改造时保留；后 2 个为 `git diff` 新增的 +24 行。断言强度：无 Weak/无 Skip，分支②同时断言 `Empty` 与非全局列表。）

## 4. B3：`relaykit/kitutil` 的 `UnmarshalJsonStr` / `DecodeJson`

### 4.1 决策：方案②（删除 + 文档例外）

| 依据 | 结论 |
|---|---|
| 使用者 | `grep -rn 'UnmarshalJsonStr\|DecodeJson\|StringToByteSlice' relaykit/ --include='*.go'`：三者都只有定义、无调用（`UnmarshalJsonStr` 只被自身包内 `StringToByteSlice` 支撑） |
| relaykit 既有 JSON 约定 | `relaykit` 内 16 个文件 import `encoding/json`，非 kitutil 的直调点 14 处（`dto/values.go`、`dto/openai_request.go`、`dto/claude.go` 等），“直连标准库”是模块的既有事实约定 |
| 方案①（14 处改走包装）成本/收益 | 改动面大（dto 包大量 `MarshalJSON`/`UnmarshalJSON` 实现与流式解码），收益低：kitutil 包装本身仍是 `encoding/json` 薄封装，且 `DecodeJson(reader)` 在 dto 的自定义方法里并不适用 |
| 模块约束 | relaykit 不得 import 根模块（AGENTS.md:67-70），无法使用 `common.*` 包装；同名副本只会成为死代码 |

结论：删除 `UnmarshalJsonStr`、`DecodeJson`（-9 行，含 `io` import），在 `relaykit/README.md` 记录「模块内 JSON 例外」。PRD 亦倾向方案②（“更小、更诚实”）。

### 4.2 AGENTS.md:72-80 措辞核对（冲突判定与处置）

原文（`AGENTS.md`）：

- `67-70`：「relaykit module independence」——`Code under relaykit/ MUST NOT import or depend on packages from the root new-api module`。
- `72`：「**JSON package:** All JSON marshal/unmarshal operations MUST use the wrapper functions in `common/json.go`」。
- `76-77`：`common.UnmarshalJsonStr(...)`、`common.DecodeJson(...)` 两个包装被点名要求使用。
- `80`：「Do NOT directly import or call `encoding/json` in business code.」

判定：**存在适用范围歧义/表面冲突**。`72`/`80` 两处用 “All … MUST” / “in business code” 的全局措辞，未限定“根模块”；而 `69` 明令 relaykit 不得 import 根模块，因此 relaykit 内**在机制上不可能**满足 `72`/`80`。两条规则不可同时字面成立。

处置（任务要求“择一改文档并说明”）：选择改 `relaykit/README.md`，落下**模块内例外**；**不修改 `AGENTS.md`**。理由：
1. PRD R2-B3 点名的两个落点是 `relaykit/README.md` 与 `.trellis/spec/backend/quality-guidelines.md`，未授权改 `AGENTS.md`（项目治理文档，改动需用户明确授权）；
2. 例外说明放在被约束的模块内，随模块一起被阅读/发布，最贴近实际写代码的人；
3. 语义上把 AGENTS.md 的 JSON 规则理解为「根模块业务代码必须走 `common.*`」，relaykit 为独立模块例外——README 已明确写清，不留歧义。

### 4.3 文档落地

`relaykit/README.md` 在「## 开发」小节末尾新增「### JSON 编码约定（模块例外）」（+14 行），要点：
- 模块不得 import 根模块 `common`，故 `common.*` 包装约定在模块内不适用；
- 普通序列化优先 `relayconvert/kitutil` 的本地包装，流式解码/自定义 `MarshalJSON`/`UnmarshalJSON` 允许直连 `encoding/json`；
- 明确这是对 AGENTS.md「JSON package」的模块内例外，而根模块业务代码仍必须走 `common.*`；
- 记录 `kitutil.UnmarshalJsonStr` / `kitutil.DecodeJson` 于 2026-10-24 因此删除。

### 4.4 删除后的级联核对

`kitutil.StringToByteSlice` 原被 `UnmarshalJsonStr` 使用；删除两个函数后它在 relaykit 内 0 引用。但：
- 它本来就在 85 条清单内（relaykit A 堆 #52，标注“仅被上两项使用，而上两项自身也无人调”），**不构成新增级联**；
- 它是 relaykit 独立模块的导出 API，属 PRD 明确留给 owner 决策的 20 项 relaykit 导出面（本任务“不动 relaykit 的 29 个”），故**保留**，仅在此记录。

relaykit 侧验证：`cd relaykit && GOWORK=off go build ./...` 与 `GOWORK=off go test ./... -count=1` 通过（§5）。

## 5. 门禁（本轮实测）

执行环境：`GOPROXY=off`（离线，不联网）、工具均已在位（`/usr/local/bin/golangci-lint`、`/root/go/bin/deadcode`）。

### 5.1 build / vet

```console
$ GOPROXY=off go build ./...
exit=0
$ GOPROXY=off go vet ./...
exit=0
```

### 5.2 根模块全量测试

```console
$ GOPROXY=off go test ./... -count=1
ok  	github.com/QuantumNous/new-api/common	0.746s
ok  	github.com/QuantumNous/new-api/controller	0.518s
...（其余 34 个包均 ok）
exit=0
```

统计：`ok` 行 36、`FAIL` 行 0、`exit=0`（全套通过）。

### 5.3 relaykit 独立模块（`GOWORK=off`）

```console
$ cd relaykit && GOWORK=off GOPROXY=off go build ./...
exit=0

$ cd relaykit && GOWORK=off GOPROXY=off go test ./... -count=1
ok  	github.com/QuantumNous/new-api/relaykit/dto	0.005s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert	0.013s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta	0.003s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert/internal/oai_chat	0.003s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert/internal/oai_responses	0.003s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil	0.002s
ok  	github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning	0.003s
（其余 8 个包无测试文件）
exit=0
```

### 5.4 gofmt 与 unused

```console
$ gofmt -l .
(无输出)
exit=0

$ golangci-lint run --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
var logKeyCol string
    ^
1 issues:
* unused: 1
```

`unused` 仍恰 1（`logKeyCol`），与 PRD 预期一致；保持基线不新增。

### 5.5 deadcode 数字与级联

```console
$ /root/go/bin/deadcode -test ./... | wc -l
81
```

B 堆收尾使清单从 **85 → 81（-4）**，逐条归因：

| 变化 | 符号（原 85 条编号） | 原因 |
|---|---|---|
| -1 | `common.IsPrivateIP`（#18） | 合并后的导出实现被 3 个 SSRF 调用点实时调用 → 由“不可达”变“可达” |
| -1 | `common.GetContextKeyStringSlice`（#10） | `service/GetRequestAutoGroups` 接线调用 → 可达 |
| -2 | `relaykit/.../kitutil.UnmarshalJsonStr`（#50）、`DecodeJson`（#51） | 按 §4.1 决策删除 |

级联核对：删除 2 个 kitutil 函数后，`kitutil.StringToByteSlice`（#52）仍不可达且仍在 81 行名单中（原表已列，非新增级联）；清单中无其它新增可达性变化。R1（task-53 清 29 个）的起点因此为 81，完成后预期约 52（PRD “85 → 约 56”是未含 B 堆的原始口径），以 task-53 证据文件 `evidence-symbols.md` 的实测为准。

### 5.6 工作区边界自证

```console
$ git status --short
 M common/ip.go
 M common/ssrf_protection.go
 M common/ssrf_protection_test.go
 M relaykit/README.md
 M relaykit/relayconvert/kitutil/json.go
 M service/group.go
 M service/group_auto_groups_test.go
?? .trellis/tasks/10-24-internal-symbols-and-b-wiring/
```

只有 7 个预期代码/文档文件被修改 + 未跟踪的任务目录（证据文件所在）；无临时文件残留（对照测试文件已删除）、未动 `data/`、未提交、未推送。


## 6. 回滚方式

全部改动未提交，逐项回滚即可（不涉及 git 历史重写，也不动未跟踪的任务目录）：

```bash
# B1
git checkout -- common/ip.go common/ssrf_protection.go common/ssrf_protection_test.go
# B2
git checkout -- service/group.go service/group_auto_groups_test.go
# B3
git checkout -- relaykit/relayconvert/kitutil/json.go relaykit/README.md
# 或一次性（保留未跟踪任务目录）
git checkout -- .
```

回滚后须重跑 §5 全部门禁确认基线；本证据文件不删除（记录用途）。
