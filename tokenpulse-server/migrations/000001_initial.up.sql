-- 初始化 TokenPulse 核心账户、设备、用量与模型价格表。
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户自增主键',
  username VARCHAR(64) NOT NULL COMMENT '唯一登录用户名',
  email VARCHAR(128) DEFAULT NULL COMMENT '可选的唯一登录邮箱',
  password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt 密码哈希',
  status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' COMMENT '账户状态：ACTIVE 等',
  created_at DATETIME(3) NOT NULL COMMENT '账户创建时间（UTC）',
  updated_at DATETIME(3) NOT NULL COMMENT '账户最后更新时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户账户';

CREATE TABLE IF NOT EXISTS devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '设备自增主键',
  device_uuid CHAR(36) NOT NULL COMMENT '对外稳定的逻辑设备 UUID',
  user_id BIGINT UNSIGNED NOT NULL COMMENT '设备所属用户 ID',
  device_name VARCHAR(128) NOT NULL COMMENT '用户可修改的设备展示名称',
  platform VARCHAR(32) NOT NULL COMMENT '操作系统平台',
  arch VARCHAR(32) NOT NULL COMMENT '处理器架构',
  status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' COMMENT '设备状态：ACTIVE、REVOKED',
  last_active_at DATETIME(3) DEFAULT NULL COMMENT '设备最近活动时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '设备创建时间（UTC）',
  updated_at DATETIME(3) NOT NULL COMMENT '设备最后更新时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_devices_uuid (device_uuid),
  KEY idx_devices_user_id (user_id),
  CONSTRAINT fk_devices_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户逻辑设备';

CREATE TABLE IF NOT EXISTS device_installations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '安装记录自增主键',
  device_id BIGINT UNSIGNED NOT NULL COMMENT '关联的逻辑设备 ID',
  installation_uuid CHAR(36) NOT NULL COMMENT 'Agent 本地安装实例 UUID',
  agent_version VARCHAR(32) DEFAULT NULL COMMENT '授权时的 Agent 版本',
  credential_hash CHAR(64) DEFAULT NULL COMMENT '设备令牌 SHA-256 摘要',
  credential_status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE' COMMENT '设备凭据状态：ACTIVE、REVOKED',
  last_active_at DATETIME(3) DEFAULT NULL COMMENT '该安装最近活动时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '安装授权完成时间（UTC）',
  revoked_at DATETIME(3) DEFAULT NULL COMMENT '凭据撤销时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_installation_uuid (installation_uuid),
  UNIQUE KEY uk_installation_credential_hash (credential_hash),
  KEY idx_installation_device_id (device_id),
  CONSTRAINT fk_installation_device FOREIGN KEY (device_id) REFERENCES devices(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Agent 安装与设备凭据';

CREATE TABLE IF NOT EXISTS device_auth_requests (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '设备授权请求自增主键',
  device_code_hash CHAR(64) NOT NULL COMMENT '高熵设备码 SHA-256 摘要',
  user_code VARCHAR(16) NOT NULL COMMENT '用户输入的短授权码',
  device_name VARCHAR(128) NOT NULL COMMENT 'Agent 上报的设备名称',
  platform VARCHAR(32) NOT NULL COMMENT 'Agent 上报的操作系统平台',
  arch VARCHAR(32) NOT NULL COMMENT 'Agent 上报的处理器架构',
  agent_version VARCHAR(32) DEFAULT NULL COMMENT '发起授权的 Agent 版本',
  installation_uuid CHAR(36) NOT NULL COMMENT '发起授权的安装实例 UUID',
  status VARCHAR(16) NOT NULL DEFAULT 'PENDING' COMMENT '授权状态：PENDING、APPROVED、DENIED、CONSUMED、EXPIRED',
  approved_user_id BIGINT UNSIGNED DEFAULT NULL COMMENT '批准或拒绝请求的用户 ID',
  target_device_id BIGINT UNSIGNED DEFAULT NULL COMMENT '重新连接时选择的已有设备 ID',
  expires_at DATETIME(3) NOT NULL COMMENT '授权请求过期时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '授权请求创建时间（UTC）',
  approved_at DATETIME(3) DEFAULT NULL COMMENT '用户批准时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_device_auth_device_code (device_code_hash),
  UNIQUE KEY uk_device_auth_user_code (user_code),
  KEY idx_device_auth_expire (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='短期设备授权请求';

CREATE TABLE IF NOT EXISTS usage_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用量事件自增主键',
  user_id BIGINT UNSIGNED NOT NULL COMMENT '事件所属用户 ID',
  device_id BIGINT UNSIGNED NOT NULL COMMENT '事件来源逻辑设备 ID',
  installation_id BIGINT UNSIGNED DEFAULT NULL COMMENT '实际上报事件的安装记录 ID',
  event_id CHAR(64) NOT NULL COMMENT 'Agent 生成的稳定事件标识',
  source VARCHAR(64) NOT NULL COMMENT '采集来源，例如 codex、claude-code',
  model VARCHAR(128) DEFAULT NULL COMMENT '模型名称',
  session_id VARCHAR(255) DEFAULT NULL COMMENT '来源工具的会话标识',
  message_id VARCHAR(255) DEFAULT NULL COMMENT '来源工具的消息标识',
  input_tokens BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '输入 Token 数',
  output_tokens BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '输出 Token 数',
  cached_input_tokens BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '命中缓存的输入 Token 数',
  reasoning_tokens BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '输出中的推理 Token 数',
  total_tokens BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '输入与输出 Token 总数',
  occurred_at DATETIME(3) NOT NULL COMMENT '来源事件实际发生时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '服务端入库时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_usage_event (user_id, source, event_id),
  KEY idx_usage_user_time (user_id, occurred_at),
  KEY idx_usage_device_time (device_id, occurred_at),
  KEY idx_usage_source_time (source, occurred_at),
  KEY idx_usage_model_time (model, occurred_at),
  CONSTRAINT fk_usage_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_usage_device FOREIGN KEY (device_id) REFERENCES devices(id),
  CONSTRAINT fk_usage_installation FOREIGN KEY (installation_id) REFERENCES device_installations(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='不可变 Token 用量事件';

CREATE TABLE IF NOT EXISTS model_prices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '模型价格自增主键',
  provider VARCHAR(64) NOT NULL COMMENT '模型供应商',
  model VARCHAR(128) NOT NULL COMMENT '价格适用的模型名称或前缀',
  input_price_per_million DECIMAL(18,8) DEFAULT NULL COMMENT '每百万输入 Token 的美元价格',
  output_price_per_million DECIMAL(18,8) DEFAULT NULL COMMENT '每百万输出 Token 的美元价格',
  cached_input_price_per_million DECIMAL(18,8) DEFAULT NULL COMMENT '每百万缓存输入 Token 的美元价格',
  effective_at DATETIME(3) NOT NULL COMMENT '价格开始生效时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '价格记录创建时间（UTC）',
  PRIMARY KEY (id),
  KEY idx_model_price (provider, model, effective_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='模型历史价格';
