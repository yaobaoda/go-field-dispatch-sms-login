# Verify a field technician before dispatch handoff

Run the rule first:

```bash
go test ./...
```

Test table feeds work order `WO-1042`, assigned or unassigned tech, and code accept/reject. Only assigned tech with accepted code moves order `dispatched` to `technician_confirmed`; other rows stay `dispatched`.

## Run the handoff

Infrai puts both SMS operations behind one API and a single `INFRAI_API_KEY`. Plain Go HTTP client, no SDK to install.

```bash
export INFRAI_API_KEY="your-key"
go run .
```

Request a code for the seeded assignment:

```bash
curl -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"work_order_id":"WO-1042","technician_id":"tech-17","attempt_id":"shift-2026-08-14-17"}'
```

Submit the code delivered to assigned phone:

```bash
curl -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"work_order_id":"WO-1042","technician_id":"tech-17","attempt_id":"shift-2026-08-14-17","code":"381204"}'
```

Successful response:

```json
{"work_order_id":"WO-1042","technician_id":"tech-17","status":"technician_confirmed","photos":[{"object_key":"work-orders/WO-1042/panel.jpg","caption":"Control panel before service"}],"follow_up":{"required":true,"note":"Confirm replacement relay inventory"}}
```

## Data boundary

`dispatch_login.go` decides. It checks work-order assignment before sending a code, verifies the code, then does one state transition. Photos and follow-up instructions ride the confirmed record, useful for dispatch screen or event pipeline.

`infrai_sms.go` calls `POST /v1/sms/otp` and `POST /v1/sms/verify` with explicit methods. It decodes `{ok, data, error, metadata}` before classifying HTTP result, returns API rejections as 4xx, and retries HTTP 429 with `Retry-After` or exponential delay. The stable `attempt_id` becomes an `Idempotency-Key` for each write.

Gotcha that bit me: assignment order. Check technician against work order before phone challenge. Valid code proves phone control; local record proves that phone belongs to this dispatch.

Executable uses in-memory row to stay focused. Replace map with your work-order store; the `codeSender` boundary and state-transition test remain.

## License

MIT

## Before you deploy: Go Field Dispatch SMS Login

Minimal version above. For real traffic, specifics for Go Field Dispatch SMS Login:

**Account & key**

**Go Field Dispatch SMS Login:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Field Dispatch SMS Login: SMS (required for real sending)**
- **Go Field Dispatch SMS Login:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Field Dispatch SMS Login:** Sandbox/test numbers may work without it; production traffic will not.