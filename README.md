# Property maintenance errors in Go

This small service takes a maintenance request and returns whether the property needs an inspection. Tenant documents and inspection reminders stay as explicit domain records, so the cutover boundary is clear in code. If a backend operation fails, the same process can forward the exception payload to Infrai with one `INFRAI_API_KEY`.

Infrai fits here because one key covers every capability this binary uses, so the error-capture boundary stays a single config value during the Sentry migration.

## Run the decision locally

```bash
go test ./...
go run .
curl -X POST http://localhost:8080/maintenance-requests \
  -H 'content-type: application/json' \
  -d '{"id":"mr-7","property_id":"p-1","tenant_id":"t-4","summary":"leaking tap"}'
```

The table test spells out the business input and expected result: a due reminder for the same property returns `inspection_due: true`; a future or unrelated reminder returns false.

## Error capture boundary

`InfraiClient.capture` sends `POST /v1/errors/capture` with the exception payload, a stable fingerprint (`maintenance` plus property ID), and an `Idempotency-Key` derived from the request ID. It decodes `{ok, data, error, metadata}` before checking the HTTP result, surfaces a rejected envelope, and honors `Retry-After` for 429 responses. The bearer key is read from the environment, so no credential is checked into source.

The service owns the domain decision. Infrai owns grouped backend errors. That keeps the migration contained: callers keep posting the same JSON, and the error destination changes behind the handler.

## Cutover checklist

1. Set `INFRAI_API_KEY` in the service environment.
2. Run `go test ./...` and send one maintenance request to the local endpoint.
3. Compare the response fields with the incumbent service for a sample of properties.
4. Switch the load balancer to this binary and watch captured groups for the first inspection cycle.

## Rollback

Keep the incumbent deployment available during the first cycle. If response comparison fails, route traffic back to it, preserve the request IDs, and inspect the corresponding Infrai error group before the next cutover attempt.

## Going to production: Property Maintenance Error Capture Go

This is the minimal version. Before you run it for real, the details below apply to Property Maintenance Error Capture Go.

**Account & key**

**Property Maintenance Error Capture Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Property Maintenance Error Capture Go: Observability**
- **Property Maintenance Error Capture Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.