# 检查用户本地部署实例（3000）：日志是否在内存

## 用户指示

「构建成功，你去检查一下，日志是不是在内存，端口是 3000…」

## 检查项

1. 实例进程/容器形态、端口、数据目录挂载；
2. **日志库是否在内存**：容器/进程 env 中的 `LOG_SQL_DSN`；磁盘上是否存在独立的日志库文件；
   接口侧证据（日志可读 = 记录在跑，但不等于在内存，需与配置文件证据交叉）；
3. 内存模式的约束是否生效：`LOG_CLEANUP_INTERVAL` / `LOG_MEMORY_MAX_ROWS` 等；
4. 结论必须区分"配置为内存"与"实际生效"，不能只看一个信号。

## 约束

- 只读为主：不重启、不改配置、不动用户数据；
- 凭据只在命令内使用，不打印、不写文件（用户给了账号，可用于接口查询）。

## Acceptance Criteria

- [ ] 给出"日志是否在内存"的明确结论 + 至少两类独立证据
- [ ] 列出关键配置（内存相关）与磁盘侧对照
- [ ] 若发现未按预期（例如仍在写磁盘），指出原因与修法

## 检查结果（2026-10-06）

**结论：日志不在内存**（两处都在磁盘上）。

| 证据 | 结果 |
|---|---|
| 容器 `new-api`（`ghcr.io/zhemed/new-api-own:latest`，version **v0.0.7**，host 网络，`:3000`）| 运行中（Up 2 minutes，restart=always，挂载 `/root/data:/data`）|
| 容器环境变量 | **没有任何 `LOG_*`**（`LOG_SQL_DSN` 未设置）|
| 日志库落点 | 主库 `/root/data/one-api.db` 内 `logs` 表**已有数据**（34 张表，`logs=1` 行）→ 日志写进了落盘的 SQLite 文件 |
| 应用文件日志 | `/root/data/logs/oneapi-20261006150607.log`（15,487 字节，目录合计 20K）→ 也在磁盘 |
| 进程启动参数 | `Entrypoint=/new-api`，无 `-log-dir=` 等覆盖 |

**原因**：内存日志是需要显式开启的（`LOG_SQL_DSN=memory`）；未设置时日志库就是主库文件，落盘。

**若要开启内存日志**（需重建容器，数据卷不受影响）：

```bash
docker rm -f new-api
docker run -d --name new-api --restart always --network host \
  -v /root/data:/data \
  -e LOG_SQL_DSN=memory \
  ghcr.io/zhemed/new-api-own:0.0.7
```

- 内存模式默认带**行数上限 20 万**与**5 分钟清理**，可用 `LOG_MEMORY_MAX_ROWS` / `LOG_CLEANUP_INTERVAL` 覆盖；
- **重启即丢**（内存日志的固有代价，符合预期）；
- 应用**文件日志**（`/data/logs/*.log`）与内存模式无关，仍在磁盘；要减少写入可加启动参数 `-log-dir=` 或按 MAINTENANCE「弱盘机器」段处理；
- 落盘模式若想控制体积，可用 `LOG_CLEANUP_INTERVAL` + `LOG_CLEANUP_RETENTION_DAYS`（磁盘模式默认关闭）。

## Acknowledgement

已按用户给的账号/密码做接口侧核对（凭据仅在命令内使用，未打印、未落盘）；数据库为只读打开（`mode=ro`）。
