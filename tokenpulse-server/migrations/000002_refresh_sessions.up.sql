-- 新增可撤销、可轮换的用户刷新会话表。
CREATE TABLE IF NOT EXISTS refresh_sessions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '刷新会话自增主键',
  user_id BIGINT UNSIGNED NOT NULL COMMENT '会话所属用户 ID',
  token_hash CHAR(64) NOT NULL COMMENT '刷新令牌 SHA-256 摘要',
  expires_at DATETIME(3) NOT NULL COMMENT '刷新令牌绝对失效时间（UTC）',
  created_at DATETIME(3) NOT NULL COMMENT '刷新会话创建时间（UTC）',
  last_used_at DATETIME(3) DEFAULT NULL COMMENT '最后轮换或撤销时间（UTC）',
  revoked_at DATETIME(3) DEFAULT NULL COMMENT '会话撤销时间（UTC）',
  PRIMARY KEY (id),
  UNIQUE KEY uk_refresh_session_token_hash (token_hash),
  KEY idx_refresh_session_user_id (user_id),
  KEY idx_refresh_session_expires_at (expires_at),
  KEY idx_refresh_session_revoked_at (revoked_at),
  CONSTRAINT fk_refresh_session_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户刷新令牌会话';
