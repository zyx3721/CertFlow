package router

// swaggerHealth godoc
// @Summary 健康检查
// @Tags Health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /api/health [get]
func swaggerHealth() {}

// swaggerCRLDistribution godoc
// @Summary 即时生成并下载公开 CRL
// @Tags PKI
// @Produce application/octet-stream
// @Param caId path string true "CA ID"
// @Success 200 {file} file
// @Failure 404 {object} errorDocResponse
// @Router /crl/{caId}.crl [get]
func swaggerCRLDistribution() {}

// swaggerOCSP godoc
// @Summary RFC 6960 OCSP 状态响应
// @Description 接收 application/ocsp-request 二进制请求，返回由签发 CA 对应的独立 OCSP Responder 身份签名的二进制响应；服务关闭时返回标准 unauthorized 响应。
// @Tags PKI
// @Accept application/ocsp-request
// @Produce application/ocsp-response
// @Param body body string true "OCSP request bytes"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /ocsp [post]
func swaggerOCSP() {}

// swaggerOCSPStatus godoc
// @Summary 查询 OCSP 状态（JSON）
// @Description 通过序列号查询适合浏览器和运维排障使用的 OCSP 状态；服务关闭时返回 unknown 与 ocspDisabled 原因。
// @Tags PKI
// @Produce json
// @Param serialNumber query string true "证书序列号"
// @Success 200 {object} ocspStatusResponse
// @Failure 400 {object} errorDocResponse
// @Router /ocsp [get]
func swaggerOCSPStatus() {}

// swaggerOCSPStatusBySerial godoc
// @Summary 查询 OCSP 状态（路径序列号）
// @Tags PKI
// @Produce json
// @Param serial path string true "证书序列号"
// @Success 200 {object} ocspStatusResponse
// @Router /ocsp/{serial} [get]
func swaggerOCSPStatusBySerial() {}

// swaggerOCSPHealth godoc
// @Summary 获取 OCSP 服务状态
// @Tags PKI
// @Produce json
// @Success 200 {object} ocspHealthResponse
// @Router /ocsp/health [get]
func swaggerOCSPHealth() {}

// swaggerLogin godoc
// @Summary 登录
// @Description LDAP 认证成功但平台未创建同名用户或该用户已禁用时，返回“用户未在平台中启用”。
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "登录参数"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/auth/login [post]
func swaggerLogin() {}

// swaggerAuthProviders godoc
// @Summary 获取登录方式
// @Tags Auth
// @Produce json
// @Success 200 {object} publicAuthProvidersResponse
// @Router /api/v1/auth/providers [get]
func swaggerAuthProviders() {}

// swaggerPublicSettings godoc
// @Summary 获取公开品牌配置
// @Description 无需登录，用于认证页和控制台读取站点名称、品牌名称与图标。
// @Tags Settings
// @Produce json
// @Success 200 {object} publicSettingsResponse
// @Router /api/v1/public/settings [get]
func swaggerPublicSettings() {}

// swaggerPasswordResetCaptcha godoc
// @Summary 获取找回密码图形验证码
// @Tags PasswordReset
// @Produce json
// @Success 200 {object} domain.PasswordResetCaptcha
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/auth/password-reset/captcha [get]
func swaggerPasswordResetCaptcha() {}

// swaggerPasswordResetVerify godoc
// @Summary 校验账号与图形验证码
// @Tags PasswordReset
// @Accept json
// @Produce json
// @Param body body passwordResetVerifyRequest true "账号和验证码"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/auth/password-reset/verify [post]
func swaggerPasswordResetVerify() {}

// swaggerPasswordResetSend godoc
// @Summary 发送找回密码邮件验证码
// @Tags PasswordReset
// @Accept json
// @Produce json
// @Param body body passwordResetSendRequest true "验证令牌与媒介"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Failure 429 {object} errorDocResponse
// @Router /api/v1/auth/password-reset/send [post]
func swaggerPasswordResetSend() {}

// swaggerPasswordResetConfirm godoc
// @Summary 确认找回密码
// @Tags PasswordReset
// @Accept json
// @Produce json
// @Param body body passwordResetConfirmRequest true "重置参数"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/auth/password-reset/confirm [post]
func swaggerPasswordResetConfirm() {}

// swaggerLogout godoc
// @Summary 注销当前会话
// @Tags Auth
// @Security BearerAuth
// @Success 204
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/auth/logout [post]
func swaggerLogout() {}

// swaggerMe godoc
// @Summary 获取当前用户
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.User
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/auth/me [get]
func swaggerMe() {}

// swaggerChangePassword godoc
// @Summary 修改当前用户密码
// @Tags Auth
// @Accept json
// @Security BearerAuth
// @Param body body changePasswordRequest true "密码参数（新密码不得与当前密码相同）"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Router /api/v1/users/change-password [post]
func swaggerChangePassword() {}

// swaggerListCAs godoc
// @Summary 获取 CA 列表
// @Tags CA
// @Produce json
// @Security BearerAuth
// @Success 200 {object} caListResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/cas [get]
func swaggerListCAs() {}

// swaggerCreateCA godoc
// @Summary 创建 CA
// @Tags CA
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createCARequest true "CA 参数"
// @Success 201 {object} domain.CA
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/cas [post]
func swaggerCreateCA() {}

// swaggerCACertificate godoc
// @Summary 获取 CA PEM 证书
// @Tags CA
// @Produce json
// @Security BearerAuth
// @Param id path string true "CA ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/cas/{id}/certificate [get]
func swaggerCACertificate() {}

// swaggerDownloadCACertificate godoc
// @Summary 下载 CA PEM 证书
// @Tags CA
// @Produce application/x-pem-file
// @Security BearerAuth
// @Param id path string true "CA ID"
// @Success 200 {file} file
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/cas/{id}/download [get]
func swaggerDownloadCACertificate() {}

// swaggerCADeletionCheck godoc
// @Summary 检查 CA 树是否可删除
// @Tags CA
// @Produce json
// @Security BearerAuth
// @Param id path string true "CA ID"
// @Success 200 {object} caDeletionCheckResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/cas/{id}/deletion-check [get]
func swaggerCADeletionCheck() {}

// swaggerDeleteCA godoc
// @Summary 递归删除未关联证书的 CA 树
// @Tags CA
// @Security BearerAuth
// @Param id path string true "CA ID"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/cas/{id} [delete]
func swaggerDeleteCA() {}

// swaggerListCertificates godoc
// @Summary 获取证书列表
// @Description 待审批或已驳回记录的序列号为 REQ:xxxxxxxx 申请编号，指纹为 16 字节的申请材料标识；已签发、已撤销或已过期记录返回冒号分隔的实际 X.509 序列号及 SHA-256 证书指纹。
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Success 200 {object} certificateListResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/certificates [get]
func swaggerListCertificates() {}

// swaggerCertificateCSRPreview godoc
// @Summary 生成证书申请 CSR 预览
// @Description 为系统生成模式创建真实 CSR 和与其匹配的私钥。私钥仅随当前申请提交，并由服务端加密保存。
// @Tags Certificates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body certificateCSRPreviewRequest true "CSR 预览参数"
// @Success 200 {object} certificateCSRPreviewResponse
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/certificates/csr-preview [post]
func swaggerCertificateCSRPreview() {}

// swaggerCertificateCSRInspect godoc
// @Summary 解析手动提供的 CSR
// @Description 校验 PKCS#10 CSR 签名，并返回 Subject、SAN 和公钥算法，供证书申请表单回填。
// @Tags Certificates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body certificateCSRInspectRequest true "CSR 内容"
// @Success 200 {object} certificateCSRInspectResponse
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/certificates/csr-inspect [post]
func swaggerCertificateCSRInspect() {}

// swaggerRequestCertificate godoc
// @Summary 提交证书申请
// @Description Subject 必须包含与通用名称一致的 CN；至少提供一个 DNS 名称或 IP 地址 SAN，国家代码为可选的两位代码；有效期为 1 至 7300 天，且不能超过签发 CA 的到期日。系统生成模式提交 CSR 和其匹配私钥，私钥将加密保存。成功后返回 REQ:xxxxxxxx 申请编号和计划到期时间，审批通过后将替换为实际 X.509 序列号与签发到期时间。
// @Tags Certificates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body certificateRequest true "证书申请"
// @Success 201 {object} domain.Certificate
// @Failure 400 {object} errorDocResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/certificates [post]
func swaggerRequestCertificate() {}

// swaggerCertificateTrend godoc
// @Summary 获取近六个月证书趋势
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Failure 401 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Router /api/v1/certificates/trend [get]
func swaggerCertificateTrend() {}

// swaggerCertificatePackage godoc
// @Summary 获取授权范围内的证书材料
// @Description 响应可能包含证书、私钥和 CSR，需具备下载证书权限。未签发申请返回 16 字节申请材料标识，已签发记录返回冒号分隔的 SHA-256 证书指纹。
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Param id path string true "证书 ID"
// @Success 200 {object} domain.Certificate
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/certificates/{id}/package [get]
func swaggerCertificatePackage() {}

// swaggerDownloadCertificate godoc
// @Summary 下载证书或被驳回申请的 CSR 材料
// @Description 已签发证书中，手动 CSR 申请返回单个 CRT 文件，系统生成模式返回包含私钥和完整证书链的 ZIP 压缩包；被驳回申请返回 CSR 文件，系统生成模式额外返回包含私钥的 ZIP 压缩包。
// @Tags Certificates
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path string true "证书 ID"
// @Success 200 {file} binary "CRT 证书或 ZIP 压缩包"
// @Failure 400 {object} errorDocResponse
// @Failure 403 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/certificates/{id}/download [get]
func swaggerDownloadCertificate() {}

// swaggerVerifyCertificate godoc
// @Summary 校验证书状态
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Param id path string true "证书 ID"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/certificates/{id}/verify [get]
func swaggerVerifyCertificate() {}

// swaggerRevokeCertificate godoc
// @Summary 撤销有效证书
// @Tags Certificates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "证书 ID"
// @Param body body revokeCertificateRequest true "撤销原因"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/certificates/{id}/revoke [post]
func swaggerRevokeCertificate() {}

// swaggerDeleteCertificate godoc
// @Summary 删除非有效状态的证书
// @Tags Certificates
// @Security BearerAuth
// @Param id path string true "证书 ID"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/certificates/{id} [delete]
func swaggerDeleteCertificate() {}

// swaggerListCRL godoc
// @Summary 获取 CRL 条目
// @Tags CRL
// @Produce json
// @Security BearerAuth
// @Param caId query string false "CA ID"
// @Param reason query string false "撤销原因"
// @Param keyword query string false "序列号、通用名称或 CA 关键词"
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页条数，最大 100" default(10)
// @Success 200 {object} crlListResponse
// @Router /api/v1/crl [get]
func swaggerListCRL() {}

// swaggerCRLMetadata godoc
// @Summary 获取 CRL 元数据（选定 CA 时返回其实际 CRL 签名算法）
// @Tags CRL
// @Produce json
// @Security BearerAuth
// @Param caId query string false "CA ID"
// @Param reason query string false "撤销原因"
// @Param keyword query string false "序列号、通用名称或 CA 关键词"
// @Success 200 {object} map[string]any
// @Router /api/v1/crl/metadata [get]
func swaggerCRLMetadata() {}

// swaggerDownloadCRL godoc
// @Summary 即时生成并下载当前筛选范围的 CRL
// @Tags CRL
// @Produce application/pkix-crl
// @Security BearerAuth
// @Param caId query string false "CA ID；省略时下载全部启用 CA 的 CRL 集合"
// @Param reason query string false "撤销原因"
// @Param keyword query string false "序列号、通用名称或 CA 关键词"
// @Success 200 {file} file
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/crl/download [get]
func swaggerDownloadCRL() {}

// swaggerListWorkflows godoc
// @Summary 获取审批列表
// @Tags Workflows
// @Produce json
// @Security BearerAuth
// @Success 200 {object} workflowListResponse
// @Router /api/v1/workflows [get]
func swaggerListWorkflows() {}

// swaggerUpdateWorkflow godoc
// @Summary 批准或驳回证书申请
// @Tags Workflows
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "审批记录 ID"
// @Param body body workflowUpdateRequest true "审批结果"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/workflows/{id} [patch]
func swaggerUpdateWorkflow() {}

// swaggerDeleteWorkflow godoc
// @Summary 删除待审批申请
// @Tags Workflows
// @Security BearerAuth
// @Param id path string true "审批记录 ID"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Router /api/v1/workflows/{id} [delete]
func swaggerDeleteWorkflow() {}

// swaggerListAudits godoc
// @Summary 获取审计日志
// @Tags Audit
// @Produce json
// @Security BearerAuth
// @Param limit query int false "返回数量"
// @Success 200 {object} auditListResponse
// @Router /api/v1/audit-logs [get]
func swaggerListAudits() {}

// swaggerGetSettings godoc
// @Summary 获取系统基础配置
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} pkiSettings
// @Router /api/v1/settings [get]
func swaggerGetSettings() {}

// swaggerSaveSettings godoc
// @Summary 保存系统基础配置
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body pkiSettings true "系统基础配置"
// @Success 200 {object} pkiSettings
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings [put]
func swaggerSaveSettings() {}

// swaggerListPermissions godoc
// @Summary 获取可分配权限
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Router /api/v1/settings/permissions [get]
func swaggerListPermissions() {}

// swaggerListNotificationChannels godoc
// @Summary 获取通知媒介配置
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Router /api/v1/settings/notifications [get]
func swaggerListNotificationChannels() {}

// swaggerSaveNotificationChannel godoc
// @Summary 更新通知媒介配置
// @Description 支持 Webhook、邮件、飞书、企业微信和钉钉媒介；审批通知内容由系统内置。Webhook 可配置 POST、PUT 或 PATCH 请求方法，以及可选的 headers JSON 对象（键和值均为字符串）。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "媒介标识"
// @Param body body notificationChannelRequest true "通知媒介参数"
// @Success 200 {object} repository.NotificationChannelSetting
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings/notifications/{id} [put]
func swaggerSaveNotificationChannel() {}

// swaggerTestNotificationChannel godoc
// @Summary 测试通知媒介
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "媒介标识"
// @Param body body notificationChannelTestRequest true "测试参数"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings/notifications/{id}/test [post]
func swaggerTestNotificationChannel() {}

// swaggerListRoles godoc
// @Summary 获取角色列表
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Router /api/v1/settings/roles [get]
func swaggerListRoles() {}

// swaggerCreateRole godoc
// @Summary 创建自定义角色
// @Description 角色标识必须以小写字母开头，只能包含小写字母、数字、点、下划线或连字符，且不可重复。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body roleRequest true "角色参数"
// @Success 201 {object} domain.Role
// @Failure 400 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/roles [post]
func swaggerCreateRole() {}

// swaggerUpdateRole godoc
// @Summary 更新自定义角色
// @Description 自定义角色创建后标识和名称不可修改；可更新描述和权限，启用状态由独立状态接口维护。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "角色 ID"
// @Param body body roleRequest true "角色参数"
// @Success 200 {object} domain.Role
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/roles/{id} [put]
func swaggerUpdateRole() {}

// swaggerSetRoleDisabled godoc
// @Summary 启用或禁用自定义角色
// @Description 内置角色不可禁用；禁用自定义角色后，该角色不再授予权限，已有用户和群组关联会保留。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "角色 ID"
// @Param body body userStatusRequest true "启用状态"
// @Success 200 {object} domain.Role
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/roles/{id}/disabled [post]
func swaggerSetRoleDisabled() {}

// swaggerDeleteRole godoc
// @Summary 删除自定义角色
// @Description 内置角色不可删除；自定义角色必须先禁用再删除。
// @Tags Settings
// @Security BearerAuth
// @Param id path string true "角色 ID"
// @Success 204
// @Failure 404 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/roles/{id} [delete]
func swaggerDeleteRole() {}

// swaggerListUserGroups godoc
// @Summary 获取用户群组列表
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Router /api/v1/settings/user-groups [get]
func swaggerListUserGroups() {}

// swaggerCreateUserGroup godoc
// @Summary 创建用户群组
// @Description 群组名称不能为空且不可重复；未指定群组角色时默认使用 viewer。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body userGroupRequest true "用户群组参数"
// @Success 201 {object} domain.UserGroup
// @Failure 400 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/user-groups [post]
func swaggerCreateUserGroup() {}

// swaggerUpdateUserGroup godoc
// @Summary 更新用户群组
// @Description 用户群组创建后名称不可修改。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户群组 ID"
// @Param body body userGroupRequest true "用户群组参数"
// @Success 200 {object} domain.UserGroup
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/user-groups/{id} [put]
func swaggerUpdateUserGroup() {}

// swaggerDeleteUserGroup godoc
// @Summary 删除用户群组
// @Description 只能删除已禁用的用户群组。
// @Tags Settings
// @Security BearerAuth
// @Param id path string true "用户群组 ID"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/user-groups/{id} [delete]
func swaggerDeleteUserGroup() {}

// swaggerListManagedUsers godoc
// @Summary 获取系统配置用户列表
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listResponse
// @Router /api/v1/settings/users [get]
func swaggerListManagedUsers() {}

// swaggerCreateManagedUser godoc
// @Summary 创建系统配置用户
// @Description 用户名和有效邮箱必填，密码至少 6 个字符；用户名不可重复；未指定角色时默认使用 viewer。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body managedUserRequest true "用户配置"
// @Success 201 {object} domain.User
// @Failure 400 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/users [post]
func swaggerCreateManagedUser() {}

// swaggerUpdateManagedUser godoc
// @Summary 更新系统配置用户
// @Description 用户创建后用户名不可修改；有效邮箱必填，新密码留空时不修改，填写时至少 6 个字符；未指定角色时默认使用 viewer。
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Param body body managedUserRequest true "用户配置"
// @Success 200 {object} domain.User
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 409 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/users/{id} [put]
func swaggerUpdateManagedUser() {}

// swaggerSetManagedUserStatus godoc
// @Summary 启用或禁用系统配置用户
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Param body body userStatusRequest true "启用状态"
// @Success 200 {object} domain.User
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/users/{id}/disabled [post]
func swaggerSetManagedUserStatus() {}

// swaggerDeleteManagedUser godoc
// @Summary 删除系统配置用户
// @Description 默认管理员和当前登录用户不可删除，其他用户必须先禁用再删除。
// @Tags Settings
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Success 204
// @Failure 400 {object} errorDocResponse
// @Failure 404 {object} errorDocResponse
// @Failure 500 {object} errorDocResponse
// @Router /api/v1/settings/users/{id} [delete]
func swaggerDeleteManagedUser() {}

// swaggerGetAuthProvider godoc
// @Summary 获取 LDAP 配置
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any
// @Router /api/v1/settings/auth-provider [get]
func swaggerGetAuthProvider() {}

// swaggerSaveAuthProvider godoc
// @Summary 保存 LDAP 配置
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body configurationRequest true "LDAP 配置"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings/auth-provider [put]
func swaggerSaveAuthProvider() {}

// swaggerTestAuthProvider godoc
// @Summary 测试 LDAP 配置
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings/auth-provider/test [post]
func swaggerTestAuthProvider() {}

// swaggerGetEmailSetting godoc
// @Summary 获取找回密码邮件配置
// @Tags Settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]any
// @Router /api/v1/settings/email [get]
func swaggerGetEmailSetting() {}

// swaggerSaveEmailSetting godoc
// @Summary 保存找回密码邮件配置
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body emailConfigurationRequest true "邮件配置"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorDocResponse
// @Router /api/v1/settings/email [put]
func swaggerSaveEmailSetting() {}

// swaggerTestEmailSetting godoc
// @Summary 发送测试邮件
// @Tags Settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body emailTestRequest true "测试收件人"
// @Success 200 {object} statusResponse
// @Failure 400 {object} errorDocResponse
// @Failure 503 {object} errorDocResponse
// @Router /api/v1/settings/email/test [post]
func swaggerTestEmailSetting() {}
