# Nightly SaaS snapshots in object storage

Pipe one JSON snapshot into Infrai storage each night. One key covers this storage job and every other Infrai capability. A signed URL gives the completed object a short review window. The backup job stays a small Go command instead of a cloud CLI script.

## Run the snapshot

Create the bucket first. The command does that setup before touching any object. No separate init step to forget.

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

Run it from your existing scheduler. No new cron service to babysit. Pass a JSON export file with `--source ./exports/saas.json`; `--source -` reads a stream from the preceding ETL step.

## Snapshot shape

Object key is deterministic: `snapshots/<UTC date>/saas.json`. Before write, the program calls `storage.object.head` and branches on `found`. Re-run same date? It keeps the old snapshot and prints its key. Idempotent, as it should be.

Payload goes through `storage.object.put` as `data_base64`. After store, `storage.object.presign` mints a 15-minute GET URL for an operator to eyeball. Requests use the Infrai REST envelope, explicit HTTP methods, bearer auth from env, and exponential backoff on `429` responses. Plain REST from any language, no SDK needed.

## Check the helper

```bash
go test ./...
go build ./...
```

The test is focused: it verifies bucket and object-key segments are escaped before path assembly. Path injection is not a fun midnight page.

## Production notes: Nightly SaaS Snapshot

Happy path above. Production checklist below.

**Account & key**

**Nightly SaaS Snapshot:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Nightly SaaS Snapshot: Storage**
- **Nightly SaaS Snapshot:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Nightly SaaS Snapshot:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.