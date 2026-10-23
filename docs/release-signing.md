# Release 资产签名（ed25519 分离签名）

> 服务于**面板内更新**（self-update）的资产校验：更新器在下载 `SHA256SUMS` 与二进制之后，
> 必须先用内置公钥验证 `SHA256SUMS.sig`，再按 `SHA256SUMS` 校验二进制；**任何一步失败都不得替换**
> 正在运行的二进制。

## 1. 签名对象与格式约定（契约，勿单方面改动）

| 项 | 约定 |
|---|---|
| 签名对象 | **`SHA256SUMS` 文件的原始字节**（Ed25519 直接对消息签名，**不预先做 SHA-256**）|
| 签名文件 | `SHA256SUMS.sig`：**一行**，`base64`（RFC 4648 标准字母表，**带 `=` 填充**）编码的 **64 字节**签名，行尾一个 `\n` |
| 公钥格式 | **raw 32 字节 ed25519 公钥的 base64**（标准字母表、带填充）|
| 生成方 | CI：`.github/workflows/release.yml` 的 `Generate SHA256SUMS` 之后的 `Sign SHA256SUMS (ed25519)` 步骤 |
| 私钥 | 仅存于 GitHub Actions secret **`UPDATE_SIGNING_KEY`**（PKCS#8 PEM，ed25519），不进仓库 |
| 覆盖关系 | `SHA256SUMS.sig` 签 `SHA256SUMS`；`SHA256SUMS` 覆盖 `new-api-linux-amd64` / `new-api-linux-arm64` |

**验签方（Go 标准库）应做的事**（示意，非仓库代码）：

```go
raw, _ := base64.StdEncoding.DecodeString(pubKeyB64)      // 32 字节
pub := ed25519.PublicKey(raw)
sig, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(sigText)) // 64 字节
ok := ed25519.Verify(pub, sha256sumsBytes, sig)           // 签的正是 SHA256SUMS 的原始字节
```

> 验签前只需 `TrimSpace` 去掉行尾换行；**不要**对 `SHA256SUMS` 做行尾归一化、CRLF 转换或重新序列化 —
> 换一个字节签名即失效（这是设计如此）。

## 2. 当前信任锚（公钥）

```
Jpj2LFm38b+U4HM/jK+4H1jJqsdiPBqw5kLx6Ihj2r8=
```

- 该值同时出现在两处，**必须保持一致**：`release.yml` 的 `UPDATE_SIGNING_PUBLIC_KEY`（CI 用它自检
  “secret 与面板信任的公钥是否匹配”）与面板（更新器内置公钥）。
- CI 在签名前会比较“secret 推导出的公钥”与这个常量；不一致直接 `::error::` 失败，**不会发布**。

## 3. 私钥存放与使用

- 存放：GitHub 仓库 `zhemed/new-api-own` → Settings → Secrets and variables → Actions →
  **`UPDATE_SIGNING_KEY`**（值 = PKCS#8 PEM 全文，含 `-----BEGIN PRIVATE KEY-----` 行）。
- 使用：只在 `release.yml` 的签名步骤读取；CI 把它写入 `$RUNNER_TEMP/update-signing-key.pem`
  （`umask 077`），步骤结束由 `trap ... EXIT` 删除；**从不 echo**。
- 本地/服务器上**不留副本**：生成本地临时文件 → 写入 secret → 立即 `rm -rf` 临时目录。

## 4. 轮换步骤（密钥泄露或定期轮换）

1. 生成新密钥对（临时目录，`umask 077`）：
   ```bash
   umask 077 && d="$(mktemp -d)" && openssl genpkey -algorithm ED25519 -out "$d/key.pem"
   ```
2. 导出新的**公钥 base64（raw 32 字节）**：
   ```bash
   ./scripts/print-ed25519-pubkey.sh "$d/key.pem"     # 输出一行 base64
   ```
3. 把新公钥写进**两处**：
   - `release.yml` 的 `UPDATE_SIGNING_PUBLIC_KEY`；
   - 面板（Go 二进制）内置公钥 —— 改 Go 代码需另行授权派工。
4. 写入新私钥到 secret（stdin，值不出现在命令行/历史里）：
   ```bash
   gh secret set UPDATE_SIGNING_KEY --repo zhemed/new-api-own < "$d/key.pem"
   gh secret list --repo zhemed/new-api-own       # 只确认名字存在
   ```
5. 删除本地临时材料并核对无残留：
   ```bash
   rm -rf "$d" && unset d
   ```
6. 发布一个版本验证：CI 会做“secret ↔ 公钥常量”自检；发布后按第 5 节命令离线验签。
7. **旧版本 Release 仍带旧签名**：面板侧应“验签失败即拒绝”，因此轮换后若要让旧版本仍可被校验，
   需要给旧 Release 重新签名（重跑对应 tag 的 release 工作流会刷新资产与签名），或维持“只更新最新版”的策略。

> 轮换期间若 CI 与新面板不同步，签名步骤会**主动失败**（公钥不匹配），不会静默发布“面板验不过”的资产。

## 5. 本地复现 / 离线验签（占位符，不含真实材料）

```bash
# 准备：把公钥 base64 落成 PEM（raw 32 字节 → 44 字节 ed25519 SPKI DER → PEM）
PUBKEY_B64='<32-byte-ed25519-public-key-base64>'
printf '%s' "$PUBKEY_B64" | base64 -d > pub.raw                     # 32 字节
printf '302a300506032b6570032100' | xxd -r -p > pub.spki.der        # SPKI 前缀
cat pub.raw >> pub.spki.der                                         # 44 字节
openssl pkey -pubin -inform DER -in pub.spki.der -out pub.pem

# 1) 验签（签名对象 = SHA256SUMS 原始字节）
base64 -d SHA256SUMS.sig > SHA256SUMS.sig.bin
openssl pkeyutl -verify -rawin -pubin -inkey pub.pem \
  -in SHA256SUMS -sigfile SHA256SUMS.sig.bin        # → Signature Verified Successfully

# 2) 按 SHA256SUMS 校验二进制
sha256sum -c SHA256SUMS

# 3) 反例：改一个字节必须失败
cp SHA256SUMS SHA256SUMS.tampered && printf 'x' >> SHA256SUMS.tampered
openssl pkeyutl -verify -rawin -pubin -inkey pub.pem \
  -in SHA256SUMS.tampered -sigfile SHA256SUMS.sig.bin   # → Signature Verification Failure
```

本地“造一对测试密钥并跑通正反例”的完整流程（不含真实密钥）：

```bash
umask 077 && d="$(mktemp -d)"
openssl genpkey -algorithm ED25519 -out "$d/k.pem"
openssl pkey -in "$d/k.pem" -pubout -out "$d/k.pub.pem"
openssl pkeyutl -sign -rawin -inkey "$d/k.pem" -in SHA256SUMS -out "$d/sig.bin"
base64 -w0 "$d/sig.bin" > "$d/SHA256SUMS.sig"
openssl pkeyutl -verify -rawin -pubin -inkey "$d/k.pub.pem" -in SHA256SUMS -sigfile "$d/sig.bin"
rm -rf "$d"
```

## 6. 排障

| 现象 | 原因与处理 |
|---|---|
| CI 报 `UPDATE_SIGNING_KEY is not set` | secret 未配置（或 fork 仓库未继承）→ 按第 4 节写入 |
| CI 报 `does not match the public key embedded in the panel` | secret 与 `UPDATE_SIGNING_PUBLIC_KEY` 不同源 → 重新导出并同步（第 4 节步骤 2–4） |
| 面板提示验签失败，但 CI 正常 | 面板内置公钥与当前 secret 不同源（轮换未同步），或 `SHA256SUMS` 被改过 |
| 本地 `openssl ... -verify` 报格式错误 | 公钥 PEM 不是 ed25519，或 `sig` 未 base64 解码（`-sigfile` 需要**二进制**签名） |
| 只想检查签名内容 | `base64 -d SHA256SUMS.sig \| xxd \| head`（64 字节；不带版本/算法前缀） |

## 7. 相关文件

- `.github/workflows/release.yml` → `Sign SHA256SUMS (ed25519)`（生成与自检）、
  `Publish release`（上传 `SHA256SUMS.sig` 并断言四件套齐全）
- `scripts/print-ed25519-pubkey.sh`（从私钥 PEM 导出 raw-32 公钥 base64，轮换用）
