-- 为刷新令牌轮换链增加 family ID，以支持重放检测后的整族撤销。
ALTER TABLE refresh_sessions ADD COLUMN family_id CHAR(36) NULL COMMENT '同一刷新令牌轮换链的 UUID' AFTER user_id;
UPDATE refresh_sessions SET family_id = UUID() WHERE family_id IS NULL;
ALTER TABLE refresh_sessions MODIFY COLUMN family_id CHAR(36) NOT NULL;
CREATE INDEX idx_refresh_session_family_id ON refresh_sessions(family_id);
