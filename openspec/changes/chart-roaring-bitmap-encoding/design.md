## Context

The chart system stores per-(dataset, entity, time-bucket) bitmaps of affected host IDs in `host_scd_data.host_bitmap` (`MEDIUMBLOB`). Today `chart.HostIDsToBlob` produces a dense bit-array sized to `(max_id / 8) + 1` bytes. The four operations Fleet performs on these blobs are `BlobAND`, `BlobOR`, `BlobANDNOT`, `BlobPopcount` (see `server/chart/blob.go`).

The dense encoding has two structural problems at scale:

1. **Per-row cost is bound to fleet-wide max host id, not to affected-host count.** Even a 1-host CVE costs ~`max_id/8` bytes if that host's id happens to be high.
2. **Avg/max blob ratio is high** because real CVEs typically include at least one host near max_id, so most rows saturate near the byte ceiling.

Measured on a real 135-host / 8k-CVE fleet: 68.5% of rows encode 1-4 hosts at an avg 186 bytes (50-200× per-bit overhead), 95.5% encode fewer than 25 hosts at near-ceiling byte length. Dense byte length is effectively a constant regardless of cardinality.

Roaring fixes both. Per-row cost scales with affected-host count for sparse bitmaps, with contiguity for dense ranges, and approaches dense-byte size only in a vanishingly rare worst case (a CVE affecting roughly 74-99% of fleet hosts spread evenly across the host-id range). In that case roaring's bitmap containers cost ~dense + tens of header bytes — sub-percent overhead, not a correctness or scale concern.

### Container-type expectations at scale

The 135-host sample fits entirely in one 65,536-bit chunk, so roaring picks array containers throughout. At 100k hosts (max_id ≈ 1.2M, ~18 chunks per row) the implementation will routinely hit all three container types:

- **Array** for sparse chunks (~majority of CVE rows by count).
- **Bitmap** for chunks with >4096 set bits — kernel CVEs, common-software CVEs at fleet scale. Fixed 8 KB per chunk; effectively equivalent to dense within that chunk.
- **Run** for chunks dominated by contiguous host-id ranges (recently-enrolled batches, full-chunk saturation).

The library auto-selects; the implementation doesn't need awareness. But the **test matrix** must cover input bitmaps that produce each container type — relying solely on small-fleet-shaped fixtures would silently leave the bitmap and run paths untested.

## Goals / Non-Goals

**Goals**

- Cut per-row blob size by ~3-4× on typical vuln data; more at 100k-host scale.
- Cut chart API wire-transfer and Go-side AND/popcount work proportionally.
- Cut cron snapshot memory peak proportionally — direct-on-roaring ops scale working memory with result cardinality, not `max_host_id`.
- No semantic change to chart output. Same buckets, same counts, identical responses.
- Schema change is a single `INSTANT` ALTER — milliseconds independent of table size.
- Lazy migration converges to all-roaring within 30 days post-deploy with no operator intervention. Open rows convert within the first cron tick (1 hour); closed rows age out via retention.

**Non-Goals**

- A user-visible knob for encoding. The format is internal.
- A dense-write fallback. New writes are always roaring; the (sub-percent) storage hit in the rare bitmap-saturated worst case is acceptable in exchange for a single uniform write path.
- 64-bit roaring; host IDs comfortably fit in 32-bit.
- Removing the dense decoder. It stays during this change as the only way to read pre-deploy closed rows still within retention. A follow-up change can delete it after production convergence is observed.

## Decisions

### `encoding_type` column discriminates the format

```sql
ALTER TABLE host_scd_data
  ADD COLUMN encoding_type TINYINT NOT NULL DEFAULT 0,
  ALGORITHM=INSTANT;
```

- `encoding_type = 0` — dense format (existing layout, unchanged byte-for-byte)
- `encoding_type = 1` — roaring format (standard `roaring.Bitmap.ToBytes()` output, post-`RunOptimize`)

Rationale:
- **No invariant fragility.** Dense rows are valid as-is; there is no requirement that "bit 0 of byte 0 is never set." Future-proof against any host-id-allocation change.
- **Indexable.** Convergence metric `SELECT COUNT(*) WHERE encoding_type = 0` is a column scan; can be indexed if it becomes hot.
- **Self-documenting.** Queries against `host_scd_data` show the format without parsing blob bytes.
- **Future-proof.** A v3 encoding bumps the column to 2 with no format-side gymnastics.
- **Cheap.** 1 byte per row; INSTANT ALTER on MySQL 8.0+ (Fleet's floor: 8.0.44).

An earlier draft of this design used a byte-0 parity discriminator that exploited the invariant that host IDs start at 1 (so bit 0 of byte 0 is never set in dense blobs, making `0x01` an unambiguous roaring tag). It was clever, but it depended on a Go-side assertion to keep the database honest, and it left no clean room for a future v3 encoding. The column is strictly simpler and removes the failure mode.

### Split storage form (`Blob`) from op form (`*roaring.Bitmap`)

```go
// Storage form
type Blob struct {
    Bytes    []byte
    Encoding uint8 // EncodingDense (0) or EncodingRoaring (1)
}

// Boundary helpers
func DecodeBitmap(b Blob) (*roaring.Bitmap, error)   // storage → op
func BitmapToBlob(rb *roaring.Bitmap) Blob           // op → storage
```

The `Blob` type appears only in code that touches `host_scd_data`. Every other piece of chart code works on `*roaring.Bitmap`. Ops (`BlobAND/OR/ANDNOT/Popcount`) take and return `*roaring.Bitmap` and have no encoding awareness.

Rationale for the split:

- **No redundant decodes in op loops.** The chart-API aggregation pattern is `merged = BlobOR(merged, r.HostBitmap)` repeated over hundreds of rows. With ops taking `Blob`, every iteration decodes `merged` (which is the accumulating result). With ops taking `*roaring.Bitmap`, `merged` stays in op form across the entire loop — O(rows) decode work instead of O(rows²).
- **No special-case popcount path.** Without a dense input to `BlobPopcount`, the `bits.OnesCount64` branch disappears. One uniform implementation.
- **Encoding awareness lives in one place.** The dispatch on `Encoding` is inside `DecodeBitmap` and nowhere else. After full convergence, deleting the dense path is a single-function change.

### Always emit roaring for new writes

```go
func BitmapToBlob(rb *roaring.Bitmap) Blob {
    rb.RunOptimize()
    return Blob{Bytes: rb.ToBytes(), Encoding: EncodingRoaring}
}

func HostIDsToBlob(ids []uint) Blob {
    return BitmapToBlob(NewBitmap(ids))
}
```

Earlier drafts of this design included a "smaller-of-two" policy that encoded both representations and picked the shorter. Analysis of when dense actually beats roaring: only when *every* chunk of a row sits in the narrow band where cardinality is just above 4096 (forcing roaring's bitmap container, ~8 KB) and the dense byte representation for that chunk is still under 8 KB. At 100k-host scale this requires a CVE affecting 74-99% of hosts spread evenly across the host-id range — kernel-tier CVEs that are uncommon, and even then the overhead is sub-percent.

The simplification is worth the rare overhead:

- One uniform write path; no branching in the encoder.
- "All new writes are roaring" is a clean invariant that holds throughout the lifetime of this change.
- After convergence, the dense decode path can be deleted entirely in a follow-up change with no policy lurking anywhere.

Empty input (`HostIDsToBlob([])`) returns `Blob{Bytes: nil, Encoding: EncodingRoaring}`. The empty case is encoding-agnostic in practice (nil bytes mean "no hosts" regardless), but tagging as Roaring keeps the invariant honest.

### Roaring-only operations

```go
func BlobAND(a, b *roaring.Bitmap) *roaring.Bitmap     // roaring.And(a, b)
func BlobOR(a, b *roaring.Bitmap) *roaring.Bitmap      // roaring.Or(a, b)
func BlobANDNOT(a, mask *roaring.Bitmap) *roaring.Bitmap // roaring.AndNot(a, mask)
func BlobPopcount(b *roaring.Bitmap) uint64            // b.GetCardinality()
```

Thin wrappers around the corresponding `RoaringBitmap/roaring` library functions, kept as the Fleet-package namespace for stability and to give Fleet a single place to attach future behavior (e.g., metric collection on op counts) without rippling through call sites. The wrappers may be inlined entirely in a future cleanup if no Fleet-specific behavior accumulates.

Rationale for keeping ops on `*roaring.Bitmap` rather than `Blob`:

- **Working memory is proportional to result size**, not to `max_host_id`. At 100k-host scale this is a ~100× reduction in transient memory per op vs the original decode-to-dense plan.
- **Skipping non-overlapping chunks is free.** For sparse CVEs touching a few chunks out of 18, most container slots are empty in both operands and the library skips them.
- **No redundant decodes.** In accumulation loops like `merged = BlobOR(merged, ...)`, `merged` stays in op form across iterations rather than being re-decoded each pass.
- **No popcount asymmetry.** Without dense inputs reaching the op layer, the `bits.OnesCount64` branch is gone.

Dense decode (when `DecodeBitmap` is called on a legacy `Blob` with `Encoding=0`) walks the dense bytes and inserts each set bit into a fresh `*roaring.Bitmap` — O(byte count) work, no large allocation. After the in-place open-row UPDATE on first cron tick, dense decode is a hot path only for the chart API's reads of closed dense rows; those age out within 30 days.

### In-place UPDATE for dense open rows in `recordSnapshot`

`recordSnapshot` today reads all open rows for the dataset and compares each existing `host_bitmap` to the new one via `bytes.Equal`. With the storage/op split, the input to `recordSnapshot` is `map[string]*roaring.Bitmap` (callers build bitmaps in op form), and existing rows are decoded to op form on read.

The path's existing safety model is **idempotency, not atomicity**. There is no enclosing transaction (the comment at `data.go:175-177` documents this). The SELECT, the close UPDATE, and the upsert INSERT are independent auto-committed statements; the design assumes stale reads via filtering by `valid_to = sentinel` and `ON DUPLICATE KEY UPDATE`.

Flow:

1. SELECT open rows (id, host_bitmap, encoding_type, valid_from).
2. For each row: `DecodeBitmap` → `*roaring.Bitmap`. Track which rows came from `encoding_type = 0`.
3. For any row sourced from dense: re-serialize the just-decoded bitmap via `BitmapToBlob` and add to a batched UPDATE:
   ```sql
   UPDATE host_scd_data
   SET host_bitmap = ?, encoding_type = 1
   WHERE id = ? AND encoding_type = 0
   ```
   The `encoding_type = 0` filter clause makes the UPDATE idempotent under stale reads — a second pass UPDATEs zero rows.
4. Compare existing (`*roaring.Bitmap`) to incoming (`*roaring.Bitmap`) via `roaring.Bitmap.Equals()`. Skip writes for unchanged entities.
5. For changed entities: close existing (UPDATE valid_to), insert new (`BitmapToBlob` → INSERT).

Why UPDATE in place rather than close+insert for the conversion:
- **Semantically truthful** — the entity's host set didn't change, only the encoding. The audit trail (valid_from / valid_to) shouldn't show a state transition.
- **Cheaper at first-tick scale** — N UPDATEs vs N (UPDATE + INSERT). Half the binlog events.
- **No bucket-boundary artifacts** — no rows with `valid_from = "now"` cluttering the time-series view.

Why `roaring.Equals()` rather than `bytes.Equal` on serialized output:
- The comparison happens in op form, not storage form — no double serialization just to compare.
- Sidesteps any non-determinism in the roaring library's serialization. A missed `RunOptimize` somewhere wouldn't cause false-positive change detections here.
- Cleaner layering: change detection is a semantic question, not a byte-level one.

Concurrency safety:
- The in-place UPDATE is idempotent via the `encoding_type = 0` filter.
- A chart-API reader hitting a row mid-conversion sees either the dense or roaring version atomically (MySQL row-level consistency); both decode to the same host set.
- A second `recordSnapshot` running concurrently (shouldn't happen, but) sees the same idempotency — at most one of them actually writes.

### Roaring serialization determinism (recommended, not load-bearing)

Every code path that produces a roaring blob — `BitmapToBlob` and by composition `HostIDsToBlob` — SHALL call `RunOptimize()` before `ToBytes()`. The library is byte-deterministic for a given set when:

1. The bitmap is built from a sorted, deduplicated input set (the library handles this for `BitmapOf(...)`).
2. `RunOptimize()` is called before `ToBytes()`.

Determinism is no longer load-bearing for correctness — change detection in `recordSnapshot` uses `roaring.Equals()` on bitmaps, not `bytes.Equal` on serialized output. But it's worth maintaining: deterministic writes are easier to reason about, allow byte-level duplicate detection in observability tooling, and prevent any future code path that does compare bytes from misfiring. The cost is one extra method call per serialization.

The test suite SHALL include a determinism test: build the same set via independent paths and assert byte-equal output from `BitmapToBlob`.

### Lazy migration via in-place UPDATE + 30-day retention

The path to a fully-roaring table:

1. `ALTER TABLE` adds the column. Existing rows logically have `encoding_type = 0`. Milliseconds.
2. Code deploys. New writes always go through `BitmapToBlob` and produce `encoding_type = 1`.
3. First cron tick after deploy: every open dense row is converted in place via batched UPDATE. Open-row population converges to all-roaring within one cron interval (1 hour).
4. Closed dense rows age out within 30 days via the existing retention cron.
5. After 30 days: full convergence. The decoder helper for dense rows can stay as a defensive safety net; the steady-state code path is roaring-only.

### Tests

- Unit tests on the four ops covering `*roaring.Bitmap` operands across sparse / medium / dense / empty / nil bit patterns. Op-layer tests need not exercise both encodings because the op layer never sees dense — that's the whole point of decode-at-boundary.
- Boundary tests on `DecodeBitmap`: dense `Blob` → `*roaring.Bitmap` whose set bits match the original dense byte layout; roaring `Blob` → equivalent `*roaring.Bitmap`. Cover edge cases: nil bytes, single-byte dense, dense spanning the 65,536-bit chunk boundary.
- Round-trip property test: random `[]uint` of various densities → `HostIDsToBlob` → `DecodeBitmap` → `ToArray()` should equal sorted-deduped input.
- **Container-type coverage**: inputs that produce array-only, bitmap-containing, and run-containing roaring payloads, plus inputs that span multiple 65,536-bit chunks. Run the full op set over these fixtures.
- **Determinism**: same set built via independent paths (`HostIDsToBlob(ids)`, `BitmapToBlob(DecodeBitmap(denseBlob))`, `BitmapToBlob(BlobOR(empty, set))`) produces byte-equal output from `BitmapToBlob`. Catches any missed `RunOptimize()` call.
- `recordSnapshot` test: a dense open row is converted in place on first cron tick with no spurious close+insert pair and no change to `valid_from` / `valid_to`. A second tick with unchanged hosts is a no-op (zero UPDATEs, zero INSERTs).
- End-to-end chart-API test exercising a chart query against rows of both encodings and asserting identical output to the all-roaring baseline.

## Risks

- **Roaring library bug surfaces in production.** Mitigated by the column-discriminated dense path remaining as a strict alternative; problematic CVEs can be diagnosed by reading `encoding_type` and comparing decoded bitmaps.
- **First-tick UPDATE volume.** ~50k UPDATEs on a 100k-host fleet's first cron tick post-deploy. Batched (aligned with `scdUpsertBatch`) and idempotent; binlog volume is manageable. Worth observing the first deploy and confirming replica lag stays bounded.
- **Transition-window query latency.** Chart queries that touch legacy dense closed rows pay decode CPU (~300-500 µs per row at 100k-host scale). A 30-day chart at 100k-host scale fetches ~15k rows; on day 1 post-deploy when nearly all closed rows in the window are still dense, that's ~4.5 seconds of extra decode CPU on top of the baseline query cost. The hit decays roughly linearly as post-deploy roaring rows replace aging-out legacy ones, reaching zero by day 31. Shorter chart ranges (1-day, 7-day) decay faster because their window doesn't span as far back. Mitigation if observed: a background closed-row backfill cron can be added in a follow-up change at any time (~100 lines, ~8 hours to converge); no rework of the lazy-migration design is required to introduce it later.

## Open Questions

- Should we add a `chart.BitmapToHostIDs(*roaring.Bitmap) []uint` exported helper for debugging / observability tools that want to inspect what's in a row? Probably yes; the library's `ToArray()` returns `[]uint32` and a thin Fleet wrapper is friendlier.
- Should `encoding_type` be indexed? Probably not initially — the metric query is run once at startup and not in any hot path. Add an index if a convergence dashboard becomes a recurring read.
- Should we drop the dense decode branch in `DecodeBitmap` after a release cycle once full convergence is observed in all production fleets? Tracked as a future cleanup change; not in this proposal.
