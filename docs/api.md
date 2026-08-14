# API

所有业务响应使用 `{ "code": 0, "message": "success", "data": ... }`，错误同时返回合理 HTTP status。Web 请求携带 HttpOnly Cookie；Agent 请求使用 `Authorization: Bearer dt_...`。

可供 Swagger UI、Redoc 或客户端生成器直接读取的规范见 [OpenAPI 3.1](openapi.yaml)。除幂等的退出接口外，浏览器的写请求还需要把 `tp_csrf` Cookie 值放入 `X-CSRF-Token` 请求头。

## Public

- `GET /health`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `POST /api/v1/device-auth/request`
- `POST /api/v1/device-auth/token`

## User authenticated

- `GET /api/v1/auth/me`
- `GET /api/v1/device-auth/info/:userCode`
- `POST /api/v1/device-auth/approve`
- `POST /api/v1/device-auth/deny`
- `GET /api/v1/devices`
- `GET /api/v1/devices/:id`
- `PATCH /api/v1/devices/:id`
- `POST /api/v1/devices/:id/revoke`
- `GET /api/v1/statistics/{summary,trend,day-detail,by-device,by-source,by-model,recent}`

统计接口支持 `startTime`、`endTime`、`deviceId`、`source`、`model` 和
`timezoneOffsetMinutes`。时间使用 RFC3339；时区偏移与浏览器
`Date#getTimezoneOffset()` 语义一致，用于“今日/本周/本月”和趋势日期分组。

## Device authenticated

- `GET /api/v1/devices/me`
- `POST /api/v1/devices/heartbeat`
- `GET /api/v1/agent/config`
- `POST /api/v1/usage/batch`（最多 500 条）
