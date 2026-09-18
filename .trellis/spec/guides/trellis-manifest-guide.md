# Trellis 平台登记与 manifest 维护指南

> 适用场景：跑过 `trellis init` / `trellis update` 之后，`.trellis/.template-hashes.json` 里的平台 key
> 消失（"平台掉登记"），或清单里冒出运行态噪声。
> 首次记录：2026-09-18，任务 `dsh-platform-reinit`。

---

## 1. 唯一判据：平台是否"已配置"由 manifest 决定

`getConfiguredPlatforms(cwd)`（trellis 0.6.17 `dist/configurators/index.js`）对每个平台只做一件事：
**检查 manifest 里有没有该平台 `configDir` 下的 key**。

- dsh 的 `configDir` 是 `.dsh` → `.dsh/**` key 为 0 时，dsh 等于未配置；
- 后果：`trellis update` 不再刷新该平台的技能文件，`trellis platforms` 不列出它，
  `trellis uninstall` 不清理它；
- **危险点**：磁盘上的技能文件都还在、会话照常可用，所以这个故障没有即时症状，
  只会静默失去托管。改完 manifest 一定回头验一次 `trellis platforms`。

```bash
trellis platforms        # 应列出 dsh：DeepSeek Harness (dsh) (dsh) — .dsh
python3 - <<'PY'
import json
d = json.load(open('.trellis/.template-hashes.json'))['hashes']
print({p: sum(k.startswith(p) for k in d) for p in ('.dsh/', '.agents/', '.trellis/')})
PY
```

## 2. 坑：重跑 `trellis init --dsh` 修不回来

`dist/utils/file-writer.js` 对**已存在且内容完全一致**的文件直接 `return false`，**不记录写盘事件**；
而 manifest 的平台段（`initializeHashes({trackedPaths})`）只收"本次真的写过"的路径。

- 症状：init 打印 `📝 Configuring …` 与 `📋 Tracking N template files`，但平台 key 数**没有变化**；
- `merge: true` 只保证已有 key 不被清空，不会把"内容一致的既有文件"补录回来；
- 结论：**平台登记一旦丢失，唯一低风险修法是直接修 manifest**（或者先把该平台文件移走再跑 init，
  代价是会重写这几十个文件）。

## 3. 修复：把平台模板 key 并回 manifest（可执行契约）

```bash
# 1) 先证明磁盘文件与当前 CLI 模板逐字节一致 —— 不一致就别用并回法，会掩盖真实差异
node --input-type=module -e "
import { collectDshTemplates } from '/usr/lib/node_modules/@mindfoldhq/trellis/dist/configurators/dsh.js';
console.log(collectDshTemplates().size);"      # dsh @0.6.17 = 47
```

```bash
# 2) 并回（key = POSIX 相对路径，value = sha256(LF 规范化内容)）
#    自检：算出的值必须等于 HEAD 里同路径的历史值 —— 内容没改过时必然相等，等就证明算法与内容都对
# 3) 验收
trellis platforms                                # 必须重新列出 dsh
```

hash 算法必须与 CLI `computeHash` 一致：`sha256(content.replace(/\r\n/g,"\n"), "utf-8")`；
JSON 结构固定 `{"__version": 2, "hashes": {…}}`、`indent=2`、**文件末尾无换行**（与 `saveHashes` 一致）。

## 4. 噪声：提交 manifest 前剔除运行态 key

`.trellis/` 的递归 walk 只排除 `workspace/`、`tasks/`、`.current-task`、`.trellis/spec/`、
`.backup-*`、`.developer`、`.version`、`.gitignore`——**不排除**这些 gitignored 路径：

- `.trellis/scripts/common/__pycache__/*.pyc`（本机实测 19 条）
- `.trellis/.runtime/**`（每个会话一个 session 文件，本机实测 4 条）

它们是机器/会话相关运行态，**不该进公开仓库的受管清单**。提交前按前缀剔除，保持既有不变量：
清单只含模板文件（HEAD 的清单即如此）。注意：下次 `trellis init` / `update` 还会重新写进去，
所以这是"每次提交前顺手清"的事，不是一次性修复。

## 5. 收尾自检

- [ ] `trellis platforms` 列出预期平台；
- [ ] manifest 平台 key 数 == `collectXxxTemplates().size`（dsh = 47）；
- [ ] `git diff --stat` 里**没有平台文件的内容改动**（修复只该动 manifest）；
- [ ] manifest 里没有 `__pycache__`、`.trellis/.runtime` key；
- [ ] 平台文件的 sha256 汇总在修复前后一致。
