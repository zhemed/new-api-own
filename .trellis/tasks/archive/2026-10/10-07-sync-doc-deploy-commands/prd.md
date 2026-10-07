# 主仓文档一致性：部署/升级/回滚命令全部带上当前日志模式

## 用户指示（2026-10-07）

「好，你负责改一下吧，要一致」

承接上一轮已指出的不一致：README.md 的**部署命令**已带四条 `LOG_*` 与 `--log-opt`，
但 **升级/回滚命令**（README.md、README.en.md、MAINTENANCE.md 共 6 处）仍是旧形态 ——
照它们重建容器会**同时丢掉内存日志模式与 docker 日志上限**。

## 要做

1. 把所有 `docker run -d --name new-api …` 命令统一为当前实例的真实形态：
   `--network host` + `--log-opt max-size=10m --log-opt max-file=3` + `-v ./data:/data`
   + 四条 `LOG_*`（`LOG_SQL_DSN=memory`、`LOG_MEMORY_MAX_BYTES=200MB`、
   `LOG_MEMORY_MAX_ROWS=200000`、`LOG_CLEANUP_RETENTION_DAYS=7`）；
2. **回滚命令**补一条注意：这四条变量依赖支持内存日志的版本，回滚到更早版本时应去掉
   （以 `git tag --contains` 查到的引入版本为准）；
3. 中文/英文两份 README 与 MAINTENANCE 表述保持一致（措辞可本地化，参数必须一致）；
4. 用脚本校验：三份文档中每个 `docker run` 命令块的参数集合**完全相同**，并与实例
   `docker inspect`（env + log-opt）逐项一致。

## 不做

- 不改 CI、不改代码、不改实例。

## Acceptance Criteria

- [ ] 三份文档所有 run 命令参数集合一致（脚本校验输出为证据）
- [ ] 与实例真实配置逐项一致（env 四条 + 两个 log-opt）
- [ ] 回滚注意事项已写明版本边界
- [ ] 提交推送；工作树干净

## 执行结果（2026-10-07）

已同步 **10 处** docker run 命令（README.md 3、README.en.md 3、MAINTENANCE.md 3、
`.trellis/spec/guides/deployment-single-method.md` 1），全部统一为实例真实形态：

```
--network host --log-opt max-size=10m --log-opt max-file=3 -v ./data:/data
-e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB
-e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7
```

补充：
- 三份文档的回滚段各加"回滚注意"（`LOG_SQL_DSN=memory` 自 `v0.0.4` 起、`LOG_MEMORY_MAX_BYTES` 自 `v0.0.8` 起，
  回滚到更早版本需去掉不认识的变量）——版本边界用 `git tag --contains` 实测确认；
- 英文 README 部署段补齐与中文版等价的四条 `LOG_*` / `--log-opt` 说明；
- 规范指南补一句"改这组参数时三处必须同步"。

## 校验证据

- 脚本解析全仓 10 处命令的参数集合 → **完全一致**（基准集含 4 个 `--*` 与 4 个 `-e LOG_*`）；
- 代码围栏配平：README.md 14、README.en.md 14、MAINTENANCE.md 44；
- `scripts/forbid-extra-deploy-methods.sh`、`scripts/forbid-encoding-json.sh` 均通过；
- 与实例 `docker inspect`（env 四条 + log-opt 两项）逐项一致。
