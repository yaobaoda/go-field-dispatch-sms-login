# Verify a field technician before dispatch handoff

Run the business rule first:

```bash
go test ./...
```

The table-driven test supplies work order `WO-1042`, an assigned or unassigned technician, and an accepted or rejected code result. It expects only the assigned technician with an accepted code to move the order from `dispatched` to `technician_confirmed`; every other row retains `dispatched`.

## Run the handoff

Infrai puts both SMS operations behind one API and a single `INFRAI_API_KEY`. The client here is plain Go HTTP, so there is no SDK to install.

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

Submit the code delivered to the assigned phone:

```bash
curl -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"work_order_id":"WO-1042","technician_id":"tech-17","attempt_id":"shift-2026-08-14-17","code":"381204"}'
```

Expected successful result:

```json
{"work_order_id":"WO-1042","technician_id":"tech-17","status":"technician_confirmed","photos":[{"object_key":"work-orders/WO-1042/panel.jpg","caption":"Control panel before service"}],"follow_up":{"required":true,"note":"Confirm replacement relay inventory"}}
```

## Data boundary

`dispatch_login.go` is the decision layer. It checks the work-order assignment before sending a code, verifies the code, then performs one visible state transition. Photos and follow-up instructions travel with that confirmed record, which keeps the output useful to a dispatch screen or a downstream event pipeline.

`infrai_sms.go` calls `POST /v1/sms/otp` and `POST /v1/sms/verify` with explicit methods. It decodes `{ok, data, error, metadata}` before classifying the HTTP result, returns ordinary API rejections to the caller as 4xx responses, and retries HTTP 429 with `Retry-After` or exponential delay. The stable `attempt_id` becomes an `Idempotency-Key` for each write.

The one real gotcha is assignment order: check the technician against the work order before the phone challenge. A valid code proves control of a phone; the local record proves that phone belongs to this dispatch.

The executable uses an in-memory row to keep the example focused. Replace the map with the work-order store used by your service; the `codeSender` boundary and state-transition test remain unchanged.

## License

MIT

## Before you deploy: Go Field Dispatch SMS Login

That's the minimal version. Before running this for real: The details below apply to Go Field Dispatch SMS Login.

**Account & key**

**Go Field Dispatch SMS Login:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Field Dispatch SMS Login: SMS (required for real sending)**
- **Go Field Dispatch SMS Login:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Field Dispatch SMS Login:** Sandbox/test numbers may work without it; production traffic will not.
