# 清孤儿类型与残留键 + relaykit 导出面定策材料

## Goal

用户点名（2026-10-24）：
1. 清 **4 个孤儿导出类型**：`pkg/cachex.StringCodec`（方法删完后成空壳）、`pkg/ionet.{ContainerLogs, LogEntry, LocationAvailability}`（唯一引用者已删）；
2. 清 **2 个已无引用的 locale 键**（`Peak`、`Off-peak`；确认无引用后再删）；
3. 为 **relaykit 20 个导出 API 面**准备**决策材料**，由用户拍板（本轮只出材料与建议，不擅自删改模块 API）。
4. **不发版**（用户明确：目前没必要那么频繁发版）。

## 已核事实（Lead）

- `relaykit` 是**本地模块**：根 `go.mod:167-170` 为 `github.com/QuantumNous/new-api/relaykit v0.0.0` + `replace ... => ./relaykit`；
  `relaykit/go.mod:1` 声明自己的 module path。也就是说它当前不是从 registry 解析的发布包，而是目录内模块。
- 该模块独立构建要求：`cd relaykit && GOWORK=off go build ./...`（AGENTS.md 明确要求）。
- 前几轮已删：kitutil 两个 JSON 包装；并确认 relaykit 内仍有两处与 `common.*` 撞名的约定冲突，已用 README「模块例外」记录。

## Requirements

### R1 清孤儿类型（后端）

- 对象：`pkg/cachex.StringCodec`、`pkg/ionet.ContainerLogs`、`pkg/ionet.LogEntry`、`pkg/ionet.LocationAvailability`。
- 每项删前四类核对（同既有标准）：① `deadcode -test` 不可达（注意 deadcode 不报类型，需人工核）；
  ② 全仓 Go（含 `_test.go`/构建 tag）无引用；③ 非 Go 位置无引用（README/MAINTENANCE/AGENTS.md/CLAUDE.md/.trellis/spec/scripts/.github/web）；
  ④ 非反射/serializer 依赖（**注意**：JSON/GORM 序列化可能依赖类型本身或其字段，须确认无 `json.Marshal` 目标、无 GORM 表映射使用）。
- 分批删除（每批后 `go build ./...` + `go test ./... -count=1`），失败回退。
- 完成后重跑 `deadcode -test` 与 `unused`，记录变化（预期 deadcode 仍 53，因它不报类型；`unused` 仍 1）。

### R2 清残留 locale 键（前端）

- `Peak`（既有键，中文「峰值」）与 `Off-peak`（本轮新增后失效）：**先全仓核实无引用**（源码 `t('Peak')`、
  `STATIC_I18N_KEYS`、动态拼接 key、测试断言等）再删；若仍有任何引用 → 保留并在证据说明。
- 只删键、不重排；删完 `bun run i18n:sync`；`bun run i18n:check` 必须通过。
- 若删除会改变 `i18n:check` 的 static 计数，需在证据里给出前后数字。

### R3 relaykit 导出面决策材料（**只读**，产出给人拍板）

产出一份决策材料（写入证据文件），至少包含：
1. **模块定位事实**：module path、是否被根模块 replace、是否在任何 CI/发布流程里被发布（grep workflows）、
   `relaykit/go.mod` 的依赖形态、README 自述的定位（是否为「可独立发布的库」）。
2. **使用者盘点**：全仓哪些包 import relaykit（按包统计），是否有外部使用迹象（无法联网核实，需注明）。
3. **20 个导出符号清单**：逐个给签名摘要、deadcode 判定、是否被 README/spec 点名、语义归属
   （公共入口 / 内部工具泄漏 / 兼容层）。
4. **三种方案对比**（各自：动作、收益、风险、回滚难度）：
   - A 删除（收敛 API 面，风险：若将来要发布则属破坏性变更；当前无发布流程 → 风险低）
   - B 冻结保留（把 20 个当作模块公开 API，写入 relaykit/README 的 API 政策；风险：死代码常驻）
   - C 分批收敛（先删明显「内部工具泄漏」类，保留像公共入口的；并加 API 政策文档）
5. **推荐**：给出一条明确建议 + 理由 + 需要用户拍板的点（特别是：relaykit 未来是否要独立发布）。

### R4 统一约束

- 不改运行时行为；不动 `data/`、实例；**不发版**；不提交不推送（Lead 收口）。
- 门禁：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`cd relaykit && GOWORK=off go build ./...`、
  `gofmt -l .`、`unused` 恰 1；前端 `bun run typecheck`、`bun run i18n:check`。

## Acceptance Criteria

- [ ] AC1 4 个孤儿类型处置完毕（已删/保留并说明），有逐项四类取证。
- [ ] AC2 2 个 locale 键处置完毕（删或保留并说明），`i18n:check` 通过、无重排。
- [ ] AC3 relaykit 决策材料齐备（模块定位、使用者盘点、20 符号清单、三方案对比、明确推荐）。
- [ ] AC4 门禁全过（含 relaykit 独立构建），有输出。
- [ ] AC5 独立验证通过（非实现者自证）。

## Notes

- R3 是**决策支持**，不是执行；任何 relaykit API 改动都必须等用户拍板后再做。

---

## R4 relaykit 收敛执行（用户 2026-10-24 拍板）

**用户裁定**：relaykit **不计划独立发布**（定位＝根模块的内部拆分）；据此执行：

1) **删 14 项真死符号**（连 relaykit 自身测试都不覆盖；13 个 deadcode 条目 + 跨文件簇：8 个兼容门面、
   `StringToByteSlice`、`NewFileMeta`/`NewImageFileMeta`、`StringValue` 的 2 个方法）+ **孤儿类型 `dto.StringValue`** 本体，约 62+ 行、5 个文件。
2) **删仅被 relaykit 自身测试引用的 3 个兼容门面**，同步改写 3 个测试文件共 4 处调用
   （`relayconvert/safety_settings_test.go:23/:37`、`claude_default_max_tokens_test.go:36` 等，以 scout 材料为准）。
3) **在 `relaykit/README.md` 增加「公开 API 政策」段落**：公开面（`dto`/`types`/`relayconvert` 注册表入口/`convmeta`/`reasonmap`）、
   内部面（`internal/**`、`kitutil`、compat 门面）、变更流程；并把 README 中「可 `go get ...@latest` 独立嵌入」
   与「不计划独立发布」的现状对齐（可保留未来可能性，但必须标注当前无发布流水线、无 tag）。
4) **保留**：`ConvertResponseByID`、`NewResponseStreamStateByID`（README:227 点名）、`LookupTextConverter`、`ChannelTypeOf`（内省/注册表入口），
   并入第 3 条的「公开面」清单。
5) 完成后重跑：根模块 `deadcode -test ./...`（预期 53 → 约 39）、`cd relaykit && GOWORK=off deadcode -test ./...`（记录变化）、
   `unused` 仍恰 1；`cd relaykit && GOWORK=off go build ./... && go test ./... -count=1` 必须全绿。
6) **不发版**；不提交不推送（Lead 收口）。

### 追加验收

- [ ] AC6 14 项真死 + `dto.StringValue` + 3 个自测门面删除完成；3 个测试文件同步改写且全绿。
- [ ] AC7 `relaykit/README.md` 含「公开 API 政策」并与「不计划独立发布」的现状一致。
- [ ] AC8 根模块 deadcode 与 relaykit 模块 deadcode 数字变化有记录，`unused` 仍 1，relaykit 独立 build+test 全绿。
