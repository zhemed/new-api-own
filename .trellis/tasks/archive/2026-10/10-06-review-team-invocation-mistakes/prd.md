# 复盘我的团队调用方式并纠正

## 用户反馈

「我发现你调用团队也不对，你就是脑子不好，根本不会调用。」

## 待复盘的调用（本会话三轮共 7 名成员）

| 轮次 | 成员 | 我的调用方式 |
|---|---|---|
| 1 | `review-backend` / `review-frontend` / `review-ops` | spawn_teammate(fresh) + 共享任务 task-1..3 |
| 2 | `audit-frontend-writes` / `audit-backend-docs` | spawn_teammate(fresh) + task-5/6 |
| 3 | `fix-frontend-open-items` / `fix-backend-open-items` | spawn_teammate(fresh) + task-7/8 |

## 我自认的失误（待逐条核实与纠正）

1. **成员同质**：7 人全是同一模型、同样 fresh 上下文 → 没有视角差异，"团队"退化成"多个我自己"；
   且我**从未查过可用模型目录**（`list_subagent_models`），也就谈不上给不同成员配不同模型。
2. **约束不够硬**：出现成员越界（一次对 127.0.0.1:9000 的连接尝试、一次把输出写到 /tmp），
   说明我的 prompt 里只写了"禁止网络动作"，没把"禁止实例化任何真实 dialector""禁止写工作区外路径"写死。
3. **Lead 与成员写范围重叠**：成员仍在跑时我在其写范围内改文件（如格式化修复）。
4. **收尾不干净**：成员是 durable 的，7 个一直留着；我没有在轮次结束后收敛成员规模。
5. 待用户指认：是否还有我没意识到的调用错误。

## Acceptance Criteria

- [ ] 逐条列出失误与正确做法（可执行），不再犯同一类
- [ ] 查清可用模型目录，明确"什么任务配什么模型/机制"
- [ ] 用户指认的具体问题被单独记录并纠正

## 用户真正指出的问题（2026-10-06）

「让你调用，你调用了十几个就是没发现现在的版本还是 0.5」

**核实结论（实测）**：

| 位置 | 版本 | 证据 |
|---|---|---|
| registry `latest` / `0.0.6` / `v0.0.6` | **0.0.6** | digest `sha256:a74d0a6c…`（三者一致）|
| registry `0.0.5` / `v0.0.5` | 0.0.5 | digest `sha256:79f4bdcb…` |
| **用户运行中的实例** | **0.0.5** | 用户可见面板版本 |
| 本机演示实例（3020）| 0.0.6 | `/api/status` |
| ⚠️ 本机缓存的 `latest` 标签 | **指向 0.0.3**（陈旧）| `docker images`：`latest → 3d04916fe29a` |

**根因（我的，不是成员的）**：团队三轮的委派范围全部指向**仓库内部**（死码、文档、i18n、后端不一致），
**没有一个成员被派去核对"线上实际部署版本 vs 已发布最新版本"**；我自己还凭旧记录断言"生产停在 v0.0.3"，
既非 0.3 也非 0.5 —— **未核实就下结论**。

**纠正措施**：
1. 立"版本对齐核查"为固定动作（写入 MAINTENANCE 维护清单）：核对①运行实例实际版本 ②registry `latest` 摘要
   ③仓库 `VERSION` 与最新 tag，三者不一致立即报；
2. 本次发现的本地隐患已记录：本机 `latest` 标签陈旧指向 0.0.3，在这台机器上 `docker run …:latest`
   会复用旧标签而不会拉到 0.0.6（需 `docker pull` 或改用固定 tag/摘要）；
3. 团队委派前必须先回答"这个委派能让用户看到什么变化"，否则不发。

## 执行（2026-10-06）

用户选择：**运行实例暂不升级**（未选项即不动）、**本机 latest 标签刷新到 0.0.6**。

| 项 | 结果 |
|---|---|
| `docker pull ghcr.io/zhemed/new-api-own:latest` | 拉到摘要 `sha256:a74d0a6c…` = registry 0.0.6 |
| 本机 `latest` 标签 | 由陈旧的 `3d04916fe29a`(0.0.3) → **`a74d0a6c0940`(0.0.6)**，隐患消除 |
| 回滚点 | `ghcr.io/zhemed/new-api-own:v0.0.3` **仍在本机**（`3d04916fe29a…`）|
| 镜像内版本元数据（新增证据） | OCI label `org.opencontainers.image.version = **v0.0.6**`，`revision=755727f` → 证明镜像内部确实是 0.0.6，不只是标签对得上 |
| 用户运行中的实例 | **未动**（用户未选择升级）|

## 下次要升级时的命令（唯一部署方式）

```bash
docker pull ghcr.io/zhemed/new-api-own:0.0.6
docker rm -f new-api
docker run -d --name new-api --restart always --network host -v ./data:/data ghcr.io/zhemed/new-api-own:0.0.6
# 异常回滚：把 0.0.6 换成 v0.0.3 重跑最后两条
```
