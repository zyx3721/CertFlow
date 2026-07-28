# CertFlow 新手证书签发与 Nginx 配置指南

本文以内部服务 `app.example.internal` 为例，说明如何在 CertFlow 中创建或导入内部 CA、签发服务端证书，并部署到 Nginx 反向代理。请将域名、IP、文件路径和上游端口替换为实际值。

## 一、先理解要完成什么

### 1.1 推荐的证书链

推荐在 CertFlow 中创建以下三层 CA：

```text
Root CA
  └── Intermediate CA
        └── Issuing CA
              └── app.example.internal 服务端证书
```

- Root CA 是内部信任锚点，应妥善保管。
- Intermediate CA 用于隔离 Root CA 与日常签发操作。
- Issuing CA 用于隔离日常签发私钥，是生产环境签发业务证书的推荐选择。
- CertFlow 允许选择任意已启用的 Root、Intermediate 或 Issuing CA 申请业务证书。若直接使用 Root 或 Intermediate CA 签发，应评估私钥使用频率、证书链长度和运维边界。
- Nginx 向客户端发送业务证书和中间链；客户端需要单独信任 Root CA。

### 1.2 与数字证书学习笔记的关系

《数字证书原理与实践》中的“根 CA → 中间 CA → 业务证书 → Nginx 部署 → 客户端导入根证书”主线，与 CertFlow 的使用流程一致。

区别在于，学习笔记使用 OpenSSL 手工生成私钥、CSR 和证书；CertFlow 将这些操作收敛为控制台和后端服务：

- CA 私钥与系统生成的业务证书私钥由后端加密保存。
- 系统生成 CSR、证书申请、审批、签发、下载和审计均在平台内完成。
- CRL 与 OCSP 地址可由系统配置写入新签发证书。

## 二、开始前的准备

### 2.1 准备域名和网络

- 为业务服务准备稳定域名，例如 `app.example.internal`。
- 确保客户端通过该域名访问 Nginx；不要用未写入证书 SAN 的 IP 或域名访问。
- 确认 Nginx 能访问上游服务，例如 `http://127.0.0.1:9000`。
- 确认已能登录 CertFlow，并拥有 CA 管理、证书申请和审批权限。
- 创建或导入 CA 需要 `ca.add` 权限；该权限同时隐含 `ca.read`，可查看上级 CA 并维护信任链。

### 2.2 规划有效期和 Subject

| 对象 | 示例名称 | 示例 Subject | 有效期建议 |
| --- | --- | --- | --- |
| Root CA | `Example Root CA` | `CN=Example Root CA,O=Example Corp,C=CN` | 最长 |
| Intermediate CA | `Example Intermediate CA` | `CN=Example Intermediate CA,O=Example Corp,C=CN` | 短于 Root CA |
| Issuing CA | `Example Issuing CA` | `CN=Example Issuing CA,O=Example Corp,C=CN` | 短于 Intermediate CA |
| 业务证书 | `app.example.internal` | `CN=app.example.internal,O=Example Corp,C=CN` | 短于 Issuing CA |

## 三、在 CertFlow 中创建或导入 CA

以下三层结构是生产环境推荐示例。若已有 CA 证书和匹配私钥，可在「CA 管理」中选择「导入 CA」替代对应创建步骤；导入同样需要 `ca.add` 权限。

### 3.1 创建 Root CA

1. 进入「CA 管理」，选择创建 CA。
2. 类型选择 `Root CA`，填写名称、密钥算法、到期日期和 Subject。
3. Root CA 不选择上级 CA，确认创建。

### 3.2 创建 Intermediate CA

1. 再次创建 CA，类型选择 `Intermediate CA`。
2. 上级 CA 选择刚创建的 Root CA。
3. 到期日期必须早于或等于 Root CA 的到期日期，确认创建。

### 3.3 创建 Issuing CA

1. 创建类型为 `Issuing CA` 的 CA。
2. 上级 CA 选择刚创建的 Intermediate CA。
3. 到期日期必须早于或等于 Intermediate CA 的到期日期，确认创建。

业务证书可选择任意已启用 CA 进行签发。本示例选择 Issuing CA，以避免 Root 和 Intermediate CA 的私钥参与日常签发。

## 四、在签发前配置 CRL 与 OCSP

若需要客户端获取撤销状态，应在申请业务证书前完成此配置；证书签发后不会自动补写 CRL 与 OCSP 扩展。

1. 进入「系统配置 - 基础配置」。
2. 启用 CRL 服务，地址填写例如 `https://pki.example.internal/crl`。
3. 启用 OCSP 服务，地址填写例如 `https://pki.example.internal/ocsp`。
4. 保存设置后再申请业务证书。

CertFlow 会在新证书中写入以下地址：

- CRL：`https://pki.example.internal/crl/<签发 CA ID>.crl`
- OCSP：`https://pki.example.internal/ocsp`

## 五、申请、审批和下载业务证书

### 5.1 推荐使用系统生成 CSR

1. 进入「证书申请」。
2. 填写 Subject、通用名称和 SAN：
   - 通用名称填写 `app.example.internal`。
   - SAN 至少包含 `app.example.internal`；需要额外域名或 IP 时逐项添加。
3. 签发 CA 可选择任意已启用的 Root、Intermediate 或 Issuing CA；本示例选择第三章创建的 Issuing CA。
4. 用途选择「服务端证书」，选择算法和有效期。
5. CSR 来源选择「系统生成」，完成预览并提交申请。

系统生成路径会保存匹配私钥。审批通过后，下载包中才会同时包含证书与私钥，适合直接部署到 Nginx。

### 5.2 审批并下载

1. 由有审批权限的用户进入「审批管理」。
2. 找到对应申请并选择「通过」；系统会立即签发证书，状态变为有效。
3. 进入「证书管理」，下载有效证书的材料包。
4. 解压 ZIP，通常包含：

```text
app.example.internal_bundle.crt  # 业务证书 + 直接签发它的 CA 证书
app.example.internal.key         # 业务证书私钥
```

> 手动提供 CSR 的模式不会把用户私钥交给 CertFlow 保存，下载时只能获得 CRT，不能直接用于本指南的 Nginx 私钥部署步骤。

### 5.3 组合 Nginx 所需的完整服务端链

下载包内的 `_bundle.crt` 包含业务证书和直接签发它的 CA 证书。若该直接签发 CA 不是 Root CA，还需要从「CA 管理」下载其到 Root CA 之间的每张中间 CA 证书，并在 Nginx 服务器上追加这些证书：

```bash
cat app.example.internal_bundle.crt example-intermediate-ca.pem > app.example.internal-fullchain.crt
```

- `app.example.internal-fullchain.crt` 的顺序应为：业务证书 → 直接签发 CA → 从下至上的各级中间 CA。
- 不要将 Root CA 放入 `ssl_certificate` 文件；Root CA 应安装到客户端受信任根证书库。
- 若业务证书由 Root CA 直接签发，下载包仍会包含 Root CA；不要直接将该 `_bundle.crt` 用于 Nginx。使用以下命令仅提取第一个叶子证书：

```bash
awk 'BEGIN { count=0 } /-----BEGIN CERTIFICATE-----/ { count++ } count == 1 { print }' app.example.internal_bundle.crt > app.example.internal-leaf.crt
```

- 若业务证书由 Intermediate CA 直接签发，则只需追加该 Intermediate CA 的上级中间证书。

## 六、部署到 Nginx 反向代理

### 6.1 安全放置证书文件

在 Nginx 服务器执行以下命令。示例假设下载和解压后的文件位于当前目录：

```bash
sudo install -d -m 700 /etc/nginx/certflow-certs
sudo install -m 600 app.example.internal.key /etc/nginx/certflow-certs/app.example.internal.key
sudo install -m 644 app.example.internal-fullchain.crt /etc/nginx/certflow-certs/app.example.internal.crt
```

若业务证书由 Root CA 直接签发，第三条命令中的源文件替换为 `app.example.internal-leaf.crt`。

### 6.2 配置 HTTPS 站点

创建 Nginx 站点配置，例如 `/etc/nginx/conf.d/app.example.internal.conf`：

```nginx
server {
    listen 80;
    server_name app.example.internal;
    return 301 https://$host$request_uri;
}

server {
    # listen 443 ssl http2;  # Nginx 1.25 以下版本写法
    listen 443 ssl;
    http2 on;
    server_name app.example.internal;

    ssl_certificate     /etc/nginx/certflow-certs/app.example.internal.crt;
    ssl_certificate_key /etc/nginx/certflow-certs/app.example.internal.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

- `server_name` 必须与证书 SAN 中的域名一致。
- `proxy_pass` 指向业务应用的实际地址；示例中的 `127.0.0.1:9000` 仅作演示。
- 若使用 Nginx 1.25 以下版本，请使用 `listen 443 ssl http2;`，不要使用独立的 `http2 on;`。

### 6.3 检查并加载 Nginx

```bash
sudo nginx -t
sudo systemctl reload nginx
```

若系统未使用 systemd，可按当前 Nginx 安装方式执行等效的 reload 命令。

## 七、让 CRL 与 OCSP 可被外部访问

若第四章配置的地址使用 `pki.example.internal`，该站点的 Nginx 还需要把公开 PKI 请求代理到 CertFlow 后端：

```nginx
location ^~ /crl/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location = /ocsp {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

这两个端点不使用浏览器会话，应保持公开可访问。若 Nginx 同时代理 CertFlow 控制台，还应保留 README 中的 `/api/`、`/swagger/` 和 `/` 代理配置。

## 八、验证结果

### 8.1 检查服务端下发的证书链

```bash
openssl s_client -connect app.example.internal:443 -servername app.example.internal -showcerts </dev/null
```

检查输出中的证书顺序是否为业务证书、直接签发 CA，以及其上级的各级中间 CA；不应包含 Root CA。

### 8.2 在客户端建立信任

内部 Root CA 不会被操作系统或浏览器默认信任。将 Root CA PEM 导入客户端的「受信任的根证书颁发机构」后，再使用 SAN 中的域名访问 `https://app.example.internal`。

### 8.3 检查撤销服务

在 CertFlow 中下载某个 CA 的 CRL，或直接访问证书中写入的 CRL 地址，确认 Nginx 能转发至后端。OCSP 使用二进制协议，建议使用支持 OCSP 的客户端或 OpenSSL 进行验证。

## 九、常见问题

| 现象 | 原因与处理方式 |
| --- | --- |
| 浏览器提示证书颁发机构无效 | 将内部 Root CA 导入客户端受信任根证书库，并确认 Nginx 下发了完整的中间证书链 |
| 浏览器提示域名不匹配 | 使用证书 SAN 中的域名访问；重新申请时补充正确的 DNS 名称或 IP |
| 下载后没有 `.key` 文件 | 使用了手动 CSR；私钥仅保留在生成 CSR 的系统中，需要在那里部署 |
| 申请时没有可选签发 CA | 创建或导入并启用任意 CA；生产环境建议启用 Issuing CA 作为日常签发 CA |
| CRL 或 OCSP 地址无法访问 | 在证书签发前保存正确的系统配置，并在 `pki.example.internal` 的 Nginx 中代理 `/crl/` 和 `/ocsp` |
| Nginx 无法读取私钥 | 检查路径、运行用户权限和私钥文件权限；私钥应避免被普通用户读取 |

## 十、续期和撤销

- 到期前重新申请并审批新证书，按第六章替换 Nginx 中的证书与私钥，然后重新加载 Nginx。
- 证书私钥泄露、主机失控或域名不再使用时，在「证书管理」撤销有效证书，并确认 CRL/OCSP 公开地址可访问。
- 撤销后不要仅依赖删除 Nginx 文件；客户端仍需要通过 CRL 或 OCSP 获取撤销状态。
