# 删除 systemd unit 并把 systemd 形态纳入闸门

## 用户定调

「一并删掉」——`new-api.service` 是裸机/systemd 部署形态，与"只有一种部署方式"冲突。

## 要做

1. 删除 `new-api.service`；
2. `scripts/forbid-extra-deploy-methods.sh` 增加 systemd 形态判定（根级 `*.service|*.timer|*.socket`、`systemd/`、`deploy/`、`etc/` 下的 unit）；
3. 规则源 `.trellis/spec/guides/deployment-single-method.md`：禁止清单加 systemd、do-not-restore 清单加该文件；
4. 文档引用同步（`docs/FILE_INVENTORY.md` 等）；
5. 复查有没有"教 systemd 部署"的文档内容残留。

## Acceptance Criteria

- [ ] 文件删除、闸门双向实测（有则该拦、无则放行）
- [ ] 规则源与文档同步，无残留
- [ ] CI 全绿
