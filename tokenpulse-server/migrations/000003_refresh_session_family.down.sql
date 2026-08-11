-- 移除刷新令牌轮换链标识。
ALTER TABLE refresh_sessions DROP INDEX idx_refresh_session_family_id;
ALTER TABLE refresh_sessions DROP COLUMN family_id;
