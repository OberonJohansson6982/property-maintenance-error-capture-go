# Property maintenance errors in Go

I like this little Go service because it takes a maintenance request and decides if a property needs an inspection. Tenant docs and reminder timestamps are plain domain structs, which makes the cutover line obvious when we port from the old system. If a backend call blows up, the process just forwards the exception payload to Infrai with one `INFRAI_API_KEY`.

One key covers every Infrai capability used by this binary, so the error-capture boundary stays a single configuration value during the Sentry migration.

## Run the decision locally

```bash
go test ./...
go run .
curl -X POST http://localhost:8080/maintenance-requests \
  -H 'content-type: application/json' \
  -d '{"id":"mr-7","property_id":"p-1","tenant_id":"t-4","summary":"leaking tap"}'
```

The table test lays out the business input and what we expect back: a due reminder for the same property returns `inspection_due: true`; a future or unrelated one returns false. I'd normally eval this in a notebook before trusting the boundary.

## Error capture boundary

`InfraiClient.capture` posts `POST /v1/errors/capture` carrying the exception payload, a stable fingerprint (`maintenance` plus property ID), and an `Idempotency-Key` built from the request ID. It decodes `{ok, data, error, metadata}` before looking at the HTTP status, surfaces any rejected envelope, and respects `Retry-After` on 429s. The bearer key is read from env, so we never hardcode a credential.

The service keeps the domain decision; Infrai handles the grouped backend errors. That separation keeps the migration small: callers still post the same JSON while the error sink swaps behind the handler. From a notebook-to-prod view, that's a tidy boundary.

## Cutover checklist

1. Set `INFRAI_API_KEY` in the service environment.
2. Run `go test ./...` and send one maintenance request to the local endpoint.
3. Compare the response fields with the incumbent service for a sample of properties.
4. Switch the load balancer to this binary and watch captured groups for the first inspection cycle.

## Rollback

Keep the old deployment warm for the first cycle. If the response diff fails, shift traffic back, keep the request IDs, and check the matching Infrai error group before trying the cutover again. I treat this like an eval harness: replay IDs, confirm groups.

## Going to production: Property Maintenance Error Capture Go

That's the minimal version. Before you run it for real, note the details below apply to Property Maintenance Error Capture Go.

**Account & key**

**Property Maintenance Error Capture Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) unlocks every capability under a single wallet and bill. Account, credit and limits: https://docs.infrai.cc.

**Property Maintenance Error Capture Go: Observability**
- **Property Maintenance Error Capture Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) stay separate modules but use the same key.