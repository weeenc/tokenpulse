-- 初始化按生效时间管理的公开 API 模型价格；来源核对于 2026-08-07：
-- https://openai.com/index/introducing-gpt-5-for-developers/
-- https://openai.com/index/introducing-gpt-5-4/
-- https://openai.com/index/gpt-5-6/
-- https://docs.anthropic.com/en/docs/about-claude/pricing
INSERT INTO model_prices
  (provider, model, input_price_per_million, output_price_per_million, cached_input_price_per_million, effective_at, created_at)
VALUES
  ('openai', 'gpt-5',       1.25000000,  10.00000000, 0.12500000, '2025-08-07 00:00:00.000', UTC_TIMESTAMP(3)),
  ('openai', 'gpt-5.2',     1.75000000,  14.00000000, 0.17500000, '2025-12-11 00:00:00.000', UTC_TIMESTAMP(3)),
  ('openai', 'gpt-5.4',     2.50000000,  15.00000000, 0.25000000, '2026-03-05 00:00:00.000', UTC_TIMESTAMP(3)),
  ('openai', 'gpt-5.6-sol', 5.00000000,  30.00000000, 0.50000000, '2026-07-09 00:00:00.000', UTC_TIMESTAMP(3)),
  ('openai', 'gpt-5.6-terra', 2.50000000, 15.00000000, 0.25000000, '2026-07-09 00:00:00.000', UTC_TIMESTAMP(3)),
  ('openai', 'gpt-5.6-luna', 1.00000000,  6.00000000, 0.10000000, '2026-07-09 00:00:00.000', UTC_TIMESTAMP(3)),
  ('anthropic', 'claude-sonnet', 3.00000000, 15.00000000, 0.30000000, '2025-05-22 00:00:00.000', UTC_TIMESTAMP(3)),
  ('anthropic', 'claude-sonnet-4', 3.00000000, 15.00000000, 0.30000000, '2025-05-22 00:00:00.000', UTC_TIMESTAMP(3)),
  ('anthropic', 'claude-opus', 15.00000000, 75.00000000, 1.50000000, '2025-05-22 00:00:00.000', UTC_TIMESTAMP(3)),
  ('anthropic', 'claude-opus-4', 15.00000000, 75.00000000, 1.50000000, '2025-05-22 00:00:00.000', UTC_TIMESTAMP(3));
