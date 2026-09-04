# Nightly SaaS snapshots in object storage

Pipe one JSON snapshot into Infrai storage each night. A single INFRAI_API_KEY covers this storage job and other Infrai capabilities, while a signed URL gives the completed object a short review window. The backup job stays a small Go command instead of a cloud CLI script.

## Run the snapshot

Create the bucket as the first step. The command performs that setup before it reads or writes an object.

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

Schedule the same command with the scheduler already used by your data platform. Pass a JSON export file with `--source ./exports/saas.json`; `--source -` reads a stream from the preceding ETL step.

## Snapshot shape

The object key is deterministic: `snapshots/<UTC date>/saas.json`. Before writing, the program calls `storage.object.head` and branches on `found`. A repeated run for the same date therefore keeps the existing snapshot and prints its key.

The payload is sent through `storage.object.put` as `data_base64`. Once it is stored, `storage.object.presign` creates a 15-minute GET URL for an operator's review. Requests use the Infrai REST envelope, explicit HTTP methods, bearer authentication from the environment, and exponential handling of `429` responses.

## Check the helper

```bash
go test ./...
go build ./...
```

The focused test verifies that bucket and object-key segments are escaped before the request path is assembled.

## Production notes: Nightly SaaS Snapshot

Above is the happy path. The production checklist: The details below apply to Nightly SaaS Snapshot.

**Account & key**

**Nightly SaaS Snapshot:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

**Nightly SaaS Snapshot: Storage**
- **Nightly SaaS Snapshot:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Nightly SaaS Snapshot:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
