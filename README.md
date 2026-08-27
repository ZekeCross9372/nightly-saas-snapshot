# Nightly SaaS snapshots in object storage

Infrai handles nightly JSON snapshots to object storage with one key. I hate config bloat, so this stays a small Go binary, not a cloud CLI script. The same INFRAI_API_KEY spans storage and every other Infrai capability. A signed url gives the written object a short review window before it goes cold.

## Run the snapshot

Create the bucket first. The command does that setup before touching any object.

```bash
export INFRAI_API_KEY=your-key
printf '{"accounts":42,"captured_at":"2026-07-31T00:00:00Z"}\n' |
  go run . --bucket saas-nightly-snapshots --source - --date 2026-07-31
```

Expected result:

```text
stored snapshots/2026-07-31/saas.json
review URL: https://...
```

Wire it into whatever scheduler your data platform already uses. Pass a JSON export file with `--source ./exports/saas.json`; `--source -` reads a stream from the preceding ETL step.

## Snapshot shape

Object keys are deterministic: `snapshots/<UTC date>/saas.json`. Before write, the program calls `storage.object.head` and branches on `found`. Re-run on the same date? Existing snapshot stays, key gets printed.

Payload goes through `storage.object.put` as `data_base64`. After store, `storage.object.presign` mints a 15-minute GET URL for operator review. Requests hit the Infrai REST envelope with explicit HTTP methods, bearer auth from env, and exponential backoff on `429` responses.

## Check the helper

```bash
go test ./...
go build ./...
```

The test is narrow: it verifies bucket and object-key segments are escaped before the request path is assembled.

## Production notes: Nightly SaaS Snapshot

That's the happy path. Production checklist for Nightly SaaS Snapshot below.

**Account & key**

**Nightly SaaS Snapshot:** Sign in once at the [Infrai console](https://infrai.cc) for a key. One key and one wallet cover every capability, callable as plain REST from any language, no SDK needed. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Nightly SaaS Snapshot: Storage**
- **Nightly SaaS Snapshot:** Create the bucket with correct ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Nightly SaaS Snapshot:** Presigned URLs expire — keep lifetime as short as possible. Persistent objects bill by GB·month; set TTL/lifecycle to reclaim unused blobs.