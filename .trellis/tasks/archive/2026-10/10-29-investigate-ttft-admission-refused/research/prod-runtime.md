# 生产机只读排障交付：用户调用模型持续 429 ttft_admission_refused

- 任务：`10-29-investigate-ttft-admission-refused` 的 task-3，生产机只读排障
- 执行方式：子会话 ssh 网络动作被本地策略闸门 S-1 拦截，已在团队消息中留痕；生产只读取证由 Lead 在主会话以兜底方式单次执行，原始记录见 `research/prod-raw.md`；本文基于该原始记录、`research/code-path.md` 与仓库只读核对交叉分析
- 授权与纪律：用户本轮明确授权生产机只读排障；未改文件、未改配置、未改数据库、未重启、未装包；未读取凭据明文；主机名、账号、密码均未写入本文件；渠道 key 未采集未落盘；base_url 为公开上游端点
- 结论口径：只读证据 + 代码证据，不含任何推测性断言不加标注

## 一、结论速览

1. 用户无法使用的直接原因：**生产网关唯一渠道的上游在故障窗口内做容量准入拒绝**。上游返回 HTTP 429，错误码 `ttft_admission_refused`，消息 `the cell is at capacity; retry shortly (after N s)`；网关把上游 429 与错误体原样回给客户端，客户端 DSH 把 429 归类为 RATE_LIMIT，按默认重试策略重试 5/5 后本轮运行失败。
2. 故障归属：**上游容量问题，不是本网关故障、不是网关限流、不是渠道配置错误、不是鉴权或余额问题**。本网关在整个窗口内进程正常、无重启循环、无 OOM 证据。
3. 放大因素：生产是**单渠道单模型**，同一模型没有备用渠道，也没有模型重定向；上游饱和时网关没有迁移目标，只能持续返回 429。网关默认 `RetryTimes=0` 且不自动禁用 429，因此既不重试也不摘除渠道，客户端每次重试都继续打同一饱和上游。
4. 范围：故障期间 `deepseek-v4.1-flash` 这一个模型不可用；由于生产只配置了这一个模型，业务上等价于聊天服务全站不可用，但管理面与网关本身正常。
5. 当前状态：故障窗口 2026-10-29 02:14-02:26Z 结束后同渠道自动恢复；2026-10-29 04:09-04:10Z 主机与容器重启后至今，`ttft` 关键字 0 次、GIN 429 0 次、200 响应正常。目前链路健康，但单渠道单点与上游间歇性容量拒绝这两个问题仍然存在。

## 二、取证方式与运行形态

### 1. 采集方式

- 采集范围：`docker ps -a`、`docker inspect` 摘要（Image、State.StartedAt、RestartCount、NetworkMode、Mounts，未打印 Env 全文）、`ss -lntp`、`docker logs --since 6h` 关键字筛取、主库 SQLite 只读查询（`file:...?mode=ro`）。
- 只读 SQL 覆盖：表清单与 schema 摘要、channels 的 id/type/name/status/priority/weight/base_url/models/model_mapping/auto_ban/setting、abilities 的模型到渠道映射、options 中重试/自动禁用/亲和/限流相关键；未查询 channels.key、tokens.key、users 口令等凭据列。
- 关键发现：生产容器环境 `LOG_SQL_DSN=memory`，即日志库为内存实现，**错误日志与消费日志不落 SQLite**。生产主库 logs 表最后一条停在 2026-10-07；因此本次 429 证据全部取自 `docker logs`，不是数据库。仓库侧印证：`model/main.go:164` 明确打印 `using in-memory SQLite as log database (usage logs are lost on restart)`。
- 证据局限：容器是 restart 而非重建，`docker logs` 保留了重启前的 02:14-02:26Z 日志；若容器被重建，该窗口证据会丢失。

### 2. 运行形态

| 项目 | 观测值 | 判断 |
|---|---|---|
| 容器 | new-api，镜像 `ghcr.io/zhemed/new-api-own:v0.0.51`，镜像创建 2026-10-24T09:12Z | 版本与仓库一致 |
| 启动 | StartedAt 2026-10-29T04:10:09Z，Restarts=0 | 无崩溃重启循环 |
| 网络 | host 网络；nginx 监听 80/443，new-api 监听 3000，sshd 22 | 前端有 nginx 反代，429 错误体来自上游，nginx 未改写 |
| 挂载 | `/root/new-api/data => /data` | 主库为 `/data/one-api.db`，6.4 MB，采集时仍在写入 |
| 主机 | Ubuntu 22.04，内核 6.8.0-138-generic；采集时 uptime 约 57 分钟，上次启动 2026-10-29T04:09Z | 与容器启动时间对齐 |
| 依赖 | 无 SQL_DSN、无 Redis；主库 SQLite，日志库内存 | 单实例单库，无横向复制 |

### 3. 时间线

| 时间 UTC | 时间本地 +08 | 事件 |
|---|---|---|
| 2026-10-29 02:14:04 | 10:14 | 第一批上游 429 失败出现，GIN 记录 `relay | <rid> | 429`，请求路径 `/v1/responses` |
| 2026-10-29 02:14-02:16 | 10:14-10:16 | 失败密集期，GIN relay 行按分钟 7、9、3 条 |
| 2026-10-29 02:23-02:26 | 10:23-10:26 | 第二波失败密集期，按分钟 2、12、7、5 条 |
| 2026-10-29 02:26:01 | 10:26 | 最后一个失败请求 ID |
| 2026-10-29 02:27 起 | 10:27 起 | 同渠道恢复，错误停止 |
| 2026-10-29 04:09 | 12:09 | 生产主机启动；原因未查证 |
| 2026-10-29 04:10:09 | 12:10 | new-api 容器启动，v0.0.51 |
| 2026-10-29 05:01-05:11 | 13:01-13:11 | 每分钟 2-12 条 200 响应，`ttft` 与 429 均为 0 |
| 2026-10-29 05:07 | 13:07 | 采集时点，主库 one-api.db 仍在写入 |

## 三、渠道、模型与容量证据

### 1. 渠道路径

- channels 表仅 1 行：id=1，type=60，name=DeepSeek，status=1，priority=0，weight=0，auto_ban=1，base_url=`https://opencode.ai/zen/go`，models=`deepseek-v4.1-flash`，model_mapping 为空，used_quota≈1.83 亿。
- type=60 的含义：`constant/channel.go:60` 定义 `ChannelTypeNewAPI = 60`，即按 New API 中转格式适配；仓库该适配器在 `relay/channel/newapi/adaptor.go`，支持 OpenAI、Claude、Gemini 请求格式与 Responses 路径转换。
- abilities 表仅 1 行：group=default，model=`deepseek-v4.1-flash`，channel_id=1，enabled=1，priority=0，weight=0。
- 结论：全生产只有一个模型、一个渠道、一条分组映射；没有备用渠道、没有同模型第二优先级、没有模型重定向、没有分组分流。

### 2. 上游指向

- base_url `https://opencode.ai/zen/go` 是公开上游端点，指向 OpenCode Zen 的 Go 套餐入口；模型为 `deepseek-v4.1-flash`。
- 渠道名 `DeepSeek` 只是本地面板命名，不代表上游是 DeepSeek 官方直连；实际上游是 OpenCode Zen。
- 渠道 status 采集时为 1，即**未被自动禁用、未被人为禁用**；结合错误日志中 429 未触发禁用逻辑，判定故障窗口内渠道保持启用态。

### 3. 网关重试与自动禁用配置

- options 表按 `%Retry%`、`%Auto%`、`%Affinity%`、`%Rate%`、`%Concurrency%` 查询只命中 `general_setting.custom_currency_exchange_rate`，说明生产未覆盖重试、自动禁用、渠道亲和、限流、并发相关配置，全部按代码默认值运行。
- 代码默认值，与 code-path 分析一致：`common.RetryTimes` 默认 0，429 属于默认可重试状态码区间但重试次数为 0，即网关不重试；`AutomaticDisableChannelEnabled` 默认关闭，默认只对 401 自动禁用，429 不会摘除渠道；网关没有基于错误率的熔断或降级。
- 因此故障期间网关行为是：收到上游 429、记两条错误日志、把 429 与错误体原样回给客户端、继续把后续请求路由到同一渠道。

## 四、错误日志证据

### 1. 关键字统计

| 统计项 | 数值 | 说明 |
|---|---|---|
| 今日 ttft 关键字日志行 | 72 行 | 每个失败请求 2 行：channel error 与 relay error |
| 唯一失败请求 ID | 36 个 | 时间跨度 02:14:04 至 02:26:01Z |
| GIN 侧 429 | 与失败请求一一对应 | `relay` 行状态码 429 |
| 上游建议等待 | 3 至 26 秒波动 | 说明是上游实时容量判断，不是固定限流窗口 |
| 重启后 ttft 关键字 | 0 次 | `--since 2026-10-29T04:10Z` |
| 重启后 GIN 429 | 0 次 | 同上 |

### 2. 典型原始行

```text
[ERR] 2026/10/29 - 02:25:34 | <rid> | channel error (channel #1, status code: 429): Upstream request failed: [ttft_admission_refused] the cell is at capacity; retry shortly (after 13 s)
[GIN] 2026/10/29 - 02:14:04 | relay | <rid> | 429 | 331.9ms | <client-ip> | POST /v1/responses
```

- 日志格式与仓库代码一致：`controller/relay.go:368` 输出 `channel error (channel #N, status code: X): ...`，`controller/relay.go:94` 输出 `relay error: ...`，正好对应每请求 2 行。
- 上游返回的是嵌套 OpenAI 错误对象：`{"error":{"message":"Upstream request failed: [ttft_admission_refused] the cell is at capacity; retry shortly (after N s)","type":"upstream_error","param":"","code":"server_error"}}`，HTTP 状态 429。
- 请求路径为 `/v1/responses`，与 code-path 判定的 DSH 客户端使用 Responses API 一致。

### 3. 请求量时间线

- GIN relay 行按分钟，UTC：02:14 7 条、02:15 9 条、02:16 3 条、02:18 3 条、02:23 2 条、02:24 12 条、02:25 7 条、02:26 5 条、02:27 4 条；合计约 52 条，其中 36 条为失败 429，其余为成功 200。
- 解读：窗口内并非 100% 失败，上游容量是间歇性紧张；但客户端一轮 5 次重试总耗时约 15 秒量级，若正好落在紧张段则 5/5 全部失败，表现为本轮运行失败；36 个失败约相当于 7 轮客户端运行。

### 4. 恢复与重启

- 02:27Z 起同渠道恢复，说明上游容量在 12 分钟内缓解，期间无需任何配置修改。
- 04:09-04:10Z 主机与容器重启，重启原因未查证，只读证据无法区分是用户处置动作、计划内维护还是异常；重启与故障窗口相隔约 1 小时 43 分，无法证明是故障处置动作。
- 重启后至采集时点，链路健康，200 响应持续。

## 五、根因链：用户为什么无法使用

```text
DSH 客户端（RATE_LIMIT 分类、默认 5 次重试、指数退避）
  │  POST /v1/responses，模型 deepseek-v4.1-flash
  ▼
生产 new-api v0.0.51
  │  middleware 选渠道：唯一映射 default/deepseek-v4.1-flash -> channel #1
  │  RetryTimes=0：不重试、不切换渠道；AutomaticDisableChannelEnabled 默认关：429 不摘除渠道
  ▼
渠道 channel #1  type=60   base_url=https://opencode.ai/zen/go
  │  转发到上游 OpenCode Zen Go
  ▼
上游返回 HTTP 429
  body 内层错误：Upstream request failed: [ttft_admission_refused] the cell is at capacity; retry shortly (after N s)
  │  envelope 与 opencode 项目 issue 中同类包装格式一致，属上游对下游失败的包装
  ▼
new-api 原样保留 message/type/code，以 429 + {"error":{...}} 回给客户端
  │  channel error 与 relay error 各记一行；不熔断、不换渠道
  ▼
DSH 客户端把 429 判为 RATE_LIMIT，重试 5/5
  │  每轮总重试时间约 15 秒，仍在容量紧张窗口内
  ▼
本轮运行失败：用户侧看到持续 429 与 RATE_LIMIT
```

逐层判定：

1. 错误码与文案产生方：`ttft_admission_refused`、`the cell is at capacity`、`Upstream request failed` 在本仓库代码零命中；网关只做错误体字段保留与透传，见 code-path 第 1、3 节。产生方在上游。
2. HTTP 429 产生方：上游。网关侧 429 可能来自自身限流，但生产未开启模型限流与全局接口限流，且错误体带上游 envelope，不是网关自身限流文案；见 code-path 第 4 节。
3. 用户看到持续失败的原因：唯一渠道的上游容量准入在窗口内反复拒绝，网关无备用渠道可切且不重试不熔断，客户端 5 次重试又都落在窗口内。
4. 网关没有放大错误，但也没有提供任何容错手段：单渠道、单模型、无重试、无熔断、错误日志不持久化。

## 六、范围判断

| 维度 | 判定 | 证据 |
|---|---|---|
| 故障范围 | 单模型 `deepseek-v4.1-flash`，因生产只有该模型，业务上等价全站聊天不可用 | channels 只有 1 行，abilities 只有 1 行 |
| 是否单渠道 | 是，唯一渠道 id=1 | channels 表 1 行 |
| 渠道是否被禁用 | 否，status=1 始终启用 | channels.status=1，options 无自动禁用覆盖 |
| 同模型是否有其他可用渠道 | 无 | abilities 无第二行 |
| 是否单个供应商问题 | 是，上游 OpenCode Zen 容量准入 | 429 错误体来自该上游 |
| 网关进程是否健康 | 是 | 容器 Restarts=0、无 OOM 证据、恢复后 200 正常 |
| 是否网关限流 | 否 | 限流默认关闭、错误体是上游 envelope |
| 是否鉴权或余额问题 | 否 | 错误是 429 容量类，非 401/403；渠道 status 正常 |
| 是否影响管理面 | 否 | 网关进程与面板继续运行 |

## 七、与 code-path 交叉验证

| 结论 | 生产证据 | 代码证据 | 一致性 |
|---|---|---|---|
| `ttft_admission_refused` 来自上游，不在本仓库 | docker logs 中 envelope 完整来自上游 429 | 全仓库零命中 | 一致 |
| 网关原样透传 429 与错误体 | GIN 429、错误体 message/type/code 完整 | `service/error.go` 保留上游字段；`relaykit/types/error.go` 原样输出 | 一致 |
| RATE_LIMIT 是客户端分类 | 网关无 RATE_LIMIT 枚举与文案 | 仅前端表单常量同名 | 一致 |
| 已重试 5/5 与秒级退避来自客户端 | 网关 RetryTimes 默认 0、无 sleep | `controller/relay.go` 重试循环无等待；默认 0 | 一致 |
| 429 不触发自动禁用、不熔断 | 故障期间与事后 channel status=1 | 自动禁用默认关，429 不在默认禁用范围 | 一致 |
| 客户端走 Responses API | 失败请求路径 `/v1/responses` | DSH 默认支持 Responses；type=60 适配器支持 Responses | 一致 |
| 网关不切换渠道 | 只有 1 个渠道可切 | 选路按分组优先级权重，单渠道无候选 | 一致 |

## 八、上游身份能确定到哪一层

- 第一层，确定：请求由生产 new-api 发往 `https://opencode.ai/zen/go`，这是渠道配置的 base_url，公开上游端点。
- 第二层，确定：该上游返回 HTTP 429，错误体是 `Upstream request failed: [ttft_admission_refused] ...`。envelope 前缀与 opencode 项目 issue 中出现的同类上游失败包装格式一致，见 code-path 第 7 节引用链接。
- 第三层，只能推断：`ttft_admission_refused` 由 OpenCode Zen 自身容量准入，还是 Zen 再下游的模型服务商产生，仅凭网关日志与公开资料无法区分。`cell` 与 `TTFT` 措辞指向首 token 时延维度的容量单元调度，但无公开文档可确证发布方。
- 第四层，确定：不是 DeepSeek 官方 API 直连，本地面板渠道名 `DeepSeek` 只是命名。

## 九、风险与不确定点

1. 单渠道单模型是单点故障：上游容量紧张时没有任何可切换目标，未来必然复发；本次是 12 分钟级爆发，历史日志显示更早日期多个时段有零星同码错误，属间歇性长期现象，频率未精确统计。
2. 重启原因未知：04:09Z 主机启动与 04:10Z 容器启动的原因没有查证；只读证据无法判断是用户处置、维护还是异常。若为异常，存在未定位的稳定性风险。
3. 错误日志不持久化：`LOG_SQL_DSN=memory` 使错误日志与消费日志在容器重建或重启时丢失，本次能取证依赖容器为 restart 且 `docker logs` 保留；这会削弱历史审计与复发追踪能力。
4. 客户端配置未直接核验：没有读取用户本机 DSH 配置，无法从配置层面证明该 provider 的 baseURL 就是本网关；但 36 个失败请求在故障窗口到达生产网关、模型名一致、路径一致、错误体回传一致，链路经过本网关的证据强。
5. 健康状态不等于根治：04:10Z 后无 429，只是上游容量恢复且期间未复发；上游再次容量紧张时症状会重现。
6. 历史分布统计不精确：原始记录中历史 ttft 分布按小时字段截取时未带日期，可能混入 10-28 及更早日期，不能作为精确频次依据。
7. 网关未透传上游建议等待时间：上游在 body 里给出 3 至 26 秒的等待建议；仓库 relay 错误路径只读响应体、未读取上游响应头，只有网关自身全局限流会写 Retry-After，见 `middleware/rate-limit.go:141`。因此客户端只能按自身固定退避重试，可能在容量仍紧张时继续失败。

## 十、处置建议

### 1. 临时规避，只读范围内不能执行，需用户或 Lead 决策后执行

1. 错峰使用或等待上游容量恢复：本次 02:27Z 已自然恢复，若当前仍在失败，先确认上游 Zen 状态。
2. 临时切换模型：若客户端支持换模型，改走其他可用模型或提供方，绕开单渠道上游。
3. 临时停用该渠道并启用备用渠道：生产当前没有备用渠道，需先新增。

### 2. 根治方向，需另开实现或运维任务并取得当轮明确授权

1. 渠道冗余：为 `deepseek-v4.1-flash` 增加第二上游渠道或第二分组，配置优先级与权重；上游 429 时至少有机会切换，避免单点。
2. 网关容错策略：按需开启 `RetryTimes` 并配置重试状态码，对 429 做有限次重试；评估是否把 429 纳入自动禁用或临时降权，避免持续打饱和上游。
3. 等待时间透传：把上游 429 body 中的建议等待时间解析并回写 `Retry-After` 响应头，让客户端退避更准确。
4. 日志持久化：把生产 `LOG_SQL_DSN=memory` 改为落盘 SQLite 或外部日志库，保留错误日志与消费日志，便于复发追踪与审计。
5. 上游可观测：为渠道补充上游身份与状态页关联，出现同类 429 时能直接定位到 OpenCode Zen 状态。

### 3. 需授权后才可执行的动作清单

- 修改生产渠道表：新增或调整渠道、优先级、权重、自动禁用与重试配置。
- 修改生产部署环境变量：`LOG_SQL_DSN` 等。
- 重启生产容器或主机。
- 新增代码改动：重试策略、Retry-After 透传、熔断或日志增强。
- 读取用户本机 DSH 配置，用于完全确认客户端 provider baseURL。

## 十一、需 Lead 决策项

1. 是否接受本报告结论：上游容量拒绝、网关无过错、单渠道单模型放大故障；据此关闭 task-3，不再补充生产取证。
2. 临时处置选哪个方向：等待上游恢复、切换客户端模型、还是新增备用渠道；其中新增渠道属于生产配置改动，需用户授权后另开任务。
3. 是否另开实现任务做网关侧增强：多渠道故障转移、429 重试策略、Retry-After 透传、日志持久化。
4. 是否需要补充只读取证：重启原因查证、nginx access log 与上游响应时间对齐、历史同码错误精确频次统计；这些需要新的生产只读授权与一次采集窗口。
5. 是否把「客户端 DSH provider baseURL 核对」作为独立事项推进，需要用户提供配置或授权读取用户本机配置。

## 十二、附录：查询语句与过滤命令

### 1. 只读数据库查询

```text
# 只读连接
# sqlite3 -readonly 直接只读打开；或 python3 用 URI 方式：sqlite3.connect('file:/root/new-api/data/one-api.db?mode=ro', uri=True)
sqlite3 -readonly -cmd ".timeout 3000" /root/new-api/data/one-api.db ".tables"
sqlite3 -readonly -cmd ".timeout 3000" /root/new-api/data/one-api.db ".schema channels"
sqlite3 -readonly -cmd ".timeout 3000" /root/new-api/data/one-api.db ".schema abilities"

# 渠道
SELECT id, type, name, status, priority, weight, auto_ban,
       base_url, models, model_mapping, test_time, response_time
FROM channels ORDER BY id;

# 模型到渠道映射
SELECT "group", model, channel_id, enabled, priority, weight, tag
FROM abilities ORDER BY model, channel_id;

# 重试/自动禁用/亲和/限流相关配置
SELECT "key", value FROM options
WHERE "key" LIKE '%Retry%' OR "key" LIKE '%Auto%' OR "key" LIKE '%Affinity%'
   OR "key" LIKE '%Rate%' OR "key" LIKE '%Concurrency%';

# 注意：生产 LOG_SQL_DSN=memory，logs 表不承载本次错误日志；
# 错误日志类型常量为 type=5（LogTypeError），type=2 是消费日志（LogTypeConsume）。
```

### 2. 运行形态与日志过滤

```text
docker ps -a --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
docker inspect -f '{{.Config.Image}} {{.State.StartedAt}} {{.RestartCount}} {{.HostConfig.NetworkMode}}' new-api
docker inspect -f '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{"\n"}}{{end}}' new-api
ss -lntp

docker logs --since 6h new-api 2>&1 | grep -E 'ttft_admission_refused|the cell is at capacity'
docker logs --since 6h new-api 2>&1 | grep -E 'GIN.*\| 429 \|'
docker logs --since 6h new-api 2>&1 | grep -E 'channel error \(channel #|relay error'
docker logs --since 6h new-api 2>&1 | grep -E 'record consume log'
```

### 3. 证据与关联文件

| 文件 | 内容 |
|---|---|
| `research/prod-raw.md` | 生产只读取证原始记录，含采集方式与授权留痕 |
| `research/code-path.md` | 代码链路分析：字符串溯源、429 传递链、重试与禁用策略 |
| `research/prod-collect.sh` | 可复用的只读一次性采集脚本，本次未在生产执行，保留作后续复采工具 |
| 本文件 | 交叉分析、结论、风险与决策项 |

### 4. 复采说明

- 若需复采，使用 `research/prod-collect.sh` 单次执行即可；该脚本已本地 dry-run 验证，只读、不写文件、不重启、不打印 Env 全文、base_url 保留后缀打码、不查询凭据列。
- 复采时建议同时保留 `docker logs` 原始时间戳与容器 StartedAt，便于区分重启前后窗口。
