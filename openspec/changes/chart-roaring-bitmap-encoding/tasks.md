## 1. Schema migration

- [x] 1.1 New migration file under `server/datastore/mysql/migrations/tables/`: `ALTER TABLE host_scd_data ADD COLUMN encoding_type TINYINT NOT NULL DEFAULT 0, ALGORITHM=INSTANT, LOCK=NONE`. Use the existing `make migration` workflow.
- [x] 1.2 Quick check: confirm `row_format` of `host_scd_data` is `Dynamic` (the default) so INSTANT applies cleanly. `SELECT row_format FROM information_schema.tables WHERE table_name = 'host_scd_data';`
- [x] 1.3 Regenerate `schema.sql` per the project workflow (`COMPOSE_PROJECT_NAME=fleet make test-schema`).
- [x] 1.4 Migration test: applyNext succeeds; column exists with expected type and default; existing rows read back with `encoding_type = 0`.

## 2. Dependency and blob package scaffolding

- [x] 2.1 Add `github.com/RoaringBitmap/roaring` to `go.mod` and `go.sum`. Confirm Go version constraint matches Fleet's (currently Go 1.26).
- [x] 2.2 Define `chart.Blob` struct (`Bytes []byte`, `Encoding uint8`) and constants `chart.EncodingDense = 0`, `chart.EncodingRoaring = 1`. Document the encoding model in a package-level comment on `server/chart/blob.go`: column-discriminated, smaller-of-two write policy, deterministic roaring serialization (RunOptimize before ToBytes).

## 3. Encoder and boundary helpers

- [x] 3.1 Add `chart.NewBitmap([]uint) *roaring.Bitmap`:
  - assert `id >= 1` for each input (sanity guard)
  - build via `roaring.BitmapOf(uint32(id)...)` (cast to uint32 for the 32-bit roaring API)
  - call `RunOptimize()` before returning
- [x] 3.2 Add `chart.BitmapToBlob(*roaring.Bitmap) chart.Blob`:
  - call `RunOptimize()` (defensive; safe to call multiple times)
  - return `Blob{Bytes: rb.ToBytes(), Encoding: EncodingRoaring}`
  - special case: a bitmap with `GetCardinality() == 0` returns `Blob{Bytes: nil, Encoding: EncodingRoaring}`
- [x] 3.3 Update `chart.HostIDsToBlob([]uint) chart.Blob` to be the convenience composition: `return BitmapToBlob(NewBitmap(ids))`.
- [x] 3.4 Unit tests covering: empty input, single bit, sparse (5 bits), medium (1k scattered), dense-contiguous (100k contiguous), dense-random (100k random). Assert each returns `Encoding = EncodingRoaring` and round-trips back to the input set via `DecodeBitmap`.

## 4. Decoder

- [x] 4.1 Add `chart.DecodeBitmap(Blob) (*roaring.Bitmap, error)`:
  - empty/nil `Bytes` → empty `*roaring.Bitmap`, no error
  - `Encoding == EncodingRoaring` → `roaring.NewBitmap()` + `FromBuffer(Bytes)`
  - `Encoding == EncodingDense` → walk bytes, for each set bit at position `i` call `rb.Add(uint32(i))`
  - unknown encoding → return error
- [x] 4.2 Unit tests for `DecodeBitmap`: nil bytes; single-byte dense; dense spanning the 65,536-bit chunk boundary; roaring round-trip; unknown encoding errors.
- [x] 4.3 Optionally export `chart.BitmapToHostIDs(*roaring.Bitmap) []uint` — thin wrapper over `ToArray()` that returns `[]uint` for callers who don't want the library's `[]uint32`.

## 5. Roaring-only operations

- [x] 5.1 Update `BlobPopcount(*roaring.Bitmap) uint64` to call `b.GetCardinality()`. No special-case for legacy dense; that's handled by `DecodeBitmap` at the boundary.
- [x] 5.2 Update `BlobAND(a, b *roaring.Bitmap) *roaring.Bitmap` to return `roaring.And(a, b)`.
- [x] 5.3 Update `BlobOR(a, b *roaring.Bitmap) *roaring.Bitmap` to return `roaring.Or(a, b)`.
- [x] 5.4 Update `BlobANDNOT(a, mask *roaring.Bitmap) *roaring.Bitmap` to return `roaring.AndNot(a, mask)`.
- [x] 5.5 Update existing call sites in `server/chart/internal/mysql/data.go`, `server/chart/internal/service/service.go`, `server/chart/datasets.go`:
  - DB read paths: load `(host_bitmap, encoding_type)` from rows, call `DecodeBitmap` to get `*roaring.Bitmap`, pass that to ops.
  - DB write paths: serialize the final `*roaring.Bitmap` via `BitmapToBlob`, write both `host_bitmap` and `encoding_type`.
  - Loops that accumulate (e.g. `merged = BlobOR(merged, ...)`): hold `merged` as `*roaring.Bitmap` across the loop; decode each row's bitmap once at iteration entry.
- [x] 5.6 Update `recordSnapshot` callers (`datasets.go:30`, `:55`): build `*roaring.Bitmap` per entity (via `NewBitmap`) and pass `map[string]*roaring.Bitmap` rather than `map[string][]byte`.

## 6. Operation-level tests

- [x] 6.1 Op-layer tests on `*roaring.Bitmap` inputs only — op layer never sees dense. For each op in {AND, OR, ANDNOT, popcount}, test sparse / medium / dense-random / empty / nil-bitmap operands. Assert correct semantic result.
- [x] 6.2 Boundary tests on `DecodeBitmap`: dense `Blob` → bitmap whose set bits match the original byte layout; roaring `Blob` → equivalent bitmap; nil bytes → empty bitmap; unknown encoding → error.
- [x] 6.3 Round-trip property test: random `[]uint` → `HostIDsToBlob` → `DecodeBitmap` → `BitmapToHostIDs` should equal sorted-deduped input.
- [x] 6.4 Idempotency on `*roaring.Bitmap` ops: `BlobAND(x, x).Equals(x)`, `BlobOR(x, x).Equals(x)`, `BlobANDNOT(x, x).IsEmpty()`.
- [x] 6.5 Container-type coverage: build fixtures that force each roaring container — array (50 scattered ids in one chunk), bitmap (>5000 ids in one chunk), run (a contiguous range of 10k ids), and multi-chunk (ids spanning ≥3 chunks across the 65,536-bit boundary). Run the full op set over these fixtures.
- [x] 6.6 Determinism: build the same set via three paths and serialize via `BitmapToBlob`:
  - `BitmapToBlob(NewBitmap(ids))`
  - `BitmapToBlob(DecodeBitmap(denseBlob))` where `denseBlob` represents the same set
  - `BitmapToBlob(BlobOR(empty, NewBitmap(ids)))`
  Assert all three byte slices are equal. Catches any missed `RunOptimize()` call.

## 7. Snapshot decode-and-convert

- [x] 7.1 Change `recordSnapshot`'s signature from `entityBitmaps map[string][]byte` to `entityBitmaps map[string]*roaring.Bitmap`. Update callers (`datasets.go`) to build bitmaps via `chart.NewBitmap`. *(landed in phase 1)*
- [x] 7.2 Update the SELECT in `recordSnapshot` (`server/chart/internal/mysql/data.go:178`) to fetch `(id, entity_id, host_bitmap, encoding_type, valid_from)`. Extend the `openRow` struct accordingly. *(landed in phase 1)*
- [x] 7.3 After the SELECT, for each row call `DecodeBitmap` to get a `*roaring.Bitmap`. Hold the decoded bitmap alongside the row in a parallel slice or as a struct field. *(landed in phase 1)*
- [ ] 7.4 *(phase 2)* For any row whose source `encoding_type = 0`: serialize the just-decoded bitmap via `BitmapToBlob` and add to a batched UPDATE list. Issue:
  ```sql
  UPDATE host_scd_data SET host_bitmap = ?, encoding_type = 1
  WHERE id = ? AND encoding_type = 0
  ```
  Batch size aligned with `scdUpsertBatch`. The `encoding_type = 0` filter clause is what makes the UPDATE idempotent under stale reads.
- [x] 7.5 Replace the existing `bytes.Equal(existing.HostBitmap, bitmap)` change-detection with `existingBitmap.Equals(incomingBitmap)` over the decoded `*roaring.Bitmap` values. *(landed in phase 1)*
- [x] 7.6 Update the upsert INSERT (`server/chart/internal/mysql/data.go:247`) to serialize each entity's `*roaring.Bitmap` via `BitmapToBlob` and write both `host_bitmap` and `encoding_type` columns. *(landed in phase 1)*
- [ ] 7.7 Test: open dense row is converted in place on first cron tick — host_bitmap and encoding_type change, valid_from and valid_to are unchanged, no new rows inserted, no rows closed.
- [ ] 7.8 Test: a second cron tick after conversion is a no-op when the host set is unchanged — zero UPDATEs (filter rejects), zero INSERTs.
- [ ] 7.9 Test: an open dense row whose host set changed on the same tick gets converted in place AND then closed+inserted (UPDATE for conversion, UPDATE for close, INSERT for new state).

## 8. End-to-end / API tests

- [ ] 8.1 Extend `server/chart/internal/mysql/data_test.go` to insert `host_scd_data` rows in both encodings and verify `GetSCDData` returns identical results regardless of source encoding.
- [ ] 8.2 Extend `server/chart/internal/service/service_test.go` for a full request → response path with mixed-encoding rows in the table.

## 9. Observability

- [ ] 9.1 Add a one-shot startup log line emitting the count of dense vs roaring rows: `SELECT COUNT(*) AS total, SUM(encoding_type = 1) AS roaring FROM host_scd_data`. Log both totals — drives lazy-migration convergence visibility on production dashboards.

## 10. Documentation

- [ ] 10.1 Update `server/chart/blob.go` package comment with the encoding model (column-discriminated, smaller-of-two write policy, deterministic roaring serialization via RunOptimize).
- [ ] 10.2 Add a CHANGELOG entry once the change ships, noting the encoding upgrade and 30-day lazy-migration window.

## 11. Lint and verification

- [ ] 11.1 `make lint-go-incremental` — 0 issues.
- [ ] 11.2 `go test ./server/chart/...` — all pass.
- [ ] 11.3 `MYSQL_TEST=1 go test ./server/chart/internal/mysql/...` — all pass.
- [ ] 11.4 Manual smoke: backfill a dev DB with mixed-encoding rows; load the dashboard; verify chart renders identically before/after the change.
