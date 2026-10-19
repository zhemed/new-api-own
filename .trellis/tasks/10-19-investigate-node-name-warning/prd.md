# 调查：NODE_NAME 警告是干什么的

## 用户问题（2026-10-19）

面板「实例」卡片显示 `ubuntu` + ⚠ 图标，悬浮提示：
「配置 NODE_NAME / 该实例正在使用自动主机名。请设置稳定且唯一的 NODE_NAME，以便进行多实例管理。
示例 NODE_NAME=new-api-master-1 / 每个实例使用不同且稳定的值，然后重启服务。」

## 只读调查项

1. `NODE_NAME` 从哪来（环境变量？设置项？）、默认值（自动主机名）如何确定；
2. 它参与哪些逻辑：多实例识别、系统任务归属/去重、日志/指标标签等；
3. ⚠ 警告的触发条件与不设置的实际风险（单实例是否也能忽略？）；
4. 设置方式与代价（需改 env + 重建容器；本实例未设 `NODE_NAME`）。

## Acceptance Criteria

- [ ] 给出代码行号级别的证据（来源、使用点、警告触发条件）
- [ ] 说明单实例场景是否真有风险
- [ ] 给出设置/不设置的建议与操作方式（不擅自改实例）

## 调查结论（2026-10-19，只读）

### 它是什么

- `common/node_identity.go:12 initNodeNameIdentity()`：`NodeName` 优先取环境变量 **`NODE_NAME`**；
  没有则**回退主机名**，并标记 `Source='hostname'`、`ShouldConfigureManually=true` → 触发面板那条 ⚠ 提示；
- 本实例实测：`NODE_NAME` **未设置**，容器 `hostname = ubuntu` → 面板显示 `ubuntu` + ⚠ ✓。

### 它被用在哪（决定了它的重要性）

| 用途 | 证据 |
|---|---|
| **实例注册表主键** | `model/system_instance.go:18` `node_name` 是 **primaryKey**；`:84` 按 last_seen 清理过期实例 |
| 审计/用量日志归属 | `model/log.go:263`、`:406`、`:468` 写入 `node_name`；`model/usedata.go:22` 有索引列 |
| 任务日志归属 | `model/task.go:111` 注释：**"发起任务的节点名，轮询结算阶段据此归属日志而非最后查询节点"** |
| 后台任务租约 | `service/system_task.go:232` `runnerID = <NodeName>-<random>`，配合租约心跳（`:614`） |

### 真实风险

- **不是单实例立刻出问题的隐患** ✓ —— 单实例下 `ubuntu` 稳定即可正常工作；
- 风险在多实例/克隆场景：`node_name` 是**主键** → 两台机器都叫 `ubuntu` 会**撞车**（实例面板只显示一台、
  日志 `node_name` 无法区分谁写的）；容器若被重建且主机名变化，日志归属也会"变节点"；
- 对**本实例**（单容器、`--network host`、主机名 `ubuntu` 稳定）：属于**建议配置**，不是必须 ✓。

### 设置方式与代价

```bash
docker run ... -e NODE_NAME=new-api-own-1 ...   # 其余参数照 README
```

代价：需要**重建容器一次**（不是 in-panel 更新能改的），会有几秒中断，且内存日志历史会被清空（已知取舍）。
