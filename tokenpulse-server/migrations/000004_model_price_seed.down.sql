-- 精确删除本迁移写入的模型价格基线。
DELETE FROM model_prices
WHERE (provider, model, effective_at) IN (
  ('openai', 'gpt-5', '2025-08-07 00:00:00.000'),
  ('openai', 'gpt-5.2', '2025-12-11 00:00:00.000'),
  ('openai', 'gpt-5.4', '2026-03-05 00:00:00.000'),
  ('openai', 'gpt-5.6-sol', '2026-07-09 00:00:00.000'),
  ('openai', 'gpt-5.6-terra', '2026-07-09 00:00:00.000'),
  ('openai', 'gpt-5.6-luna', '2026-07-09 00:00:00.000'),
  ('anthropic', 'claude-sonnet', '2025-05-22 00:00:00.000'),
  ('anthropic', 'claude-sonnet-4', '2025-05-22 00:00:00.000'),
  ('anthropic', 'claude-opus', '2025-05-22 00:00:00.000'),
  ('anthropic', 'claude-opus-4', '2025-05-22 00:00:00.000')
);
