## Why

At fleet sizes around 100k hosts, the chart system's dense bitmap encoding becomes the binding constraint on both storage and chart API latency. Each row in `host_scd_data` stores a `MEDIUMBLOB` whose length is `(max_host_id / 8) + 1` bytes. Because host IDs are monotonic and never reused, the bitmap is sized to "max host ever enrolled," not "current host count."

### Measured shape (135-host / 8k-CVE production fleet)

One week of hourly snapshots, `max_host_id = 1710` (12.7× host-id sparsity vs 135 live hosts), yielding a 215-byte natural ceiling per row. Bitmap length distribution:

| popcount | rows | % of rows | avg dense bytes |
|---|---|---|---|
| 1-4 hosts | 20,633 | 68.5% | 186 |
| 5-24 hosts | 8,131 | 27.0% | 211 |
| 25-49 hosts | 620 | 2.1% | 212 |
| 50-99 hosts | 730 | 2.4% | 213 |

**68.5% of rows encode ≤4 hosts but pay ~186 bytes — a 50-200× overhead per set bit.** 95.5% of rows encode fewer than 25 hosts but pay near the byte ceiling. The dense byte-length is essentially constant regardless of actual cardinality.

Weekly totals on this fleet: ~5.6 MB dense → ~1.1 MB roaring (all array containers, since every host_id ≤ 1710 fits in a single 65,536-bit chunk). **~80% reduction at this fleet size.**

### Projection to 100k hosts

Row count is largely host-count-independent (CVE-driven, with NVD-style spike events dominating). The same ~30k rows/week pattern applies, scaled by catalog size — for a 36k-CVE fleet, roughly **100-120k rows/week**.

**Dense** scales with `max_host_id / 8`. At 100k live hosts with similar 12× host-id sparsity, max_id ≈ 1.2M, byte-per-row ≈ 150 KB:
- ~120k rows × 150 KB ≈ **18 GB/week** worst case
- Less if fleet is younger (lower max_id) — but the trend is monotonic over time.

**Roaring** scales with set cardinality and host-id distribution across chunks. At 100k hosts the "array containers everywhere" property from the 135-host sample no longer holds — affected host IDs span ~18 chunks (1.2M / 65,536), and the per-chunk container choice depends on local cardinality:
- **Array** container (≤4096 set bits per chunk): the dominant case for low-popcount CVEs.
- **Bitmap** container (>4096 set bits per chunk): kicks in for high-impact CVEs (e.g., kernel vulns affecting much of the fleet). Fixed 8 KB per chunk.
- **Run** container: contiguous host-id ranges, e.g., a newly-enrolled batch of hosts all running the same vulnerable software. ~4 B per run.

The library picks per chunk transparently. Projected weighted average: **2-10 KB per row**, depending on whether per-CVE popcount scales linearly with fleet size or stays roughly absolute (most CVEs are tied to specific software versions, not fleet headcount).

- ~120k rows × ~6 KB ≈ **~700 MB/week**

**Storage reduction at scale: ~20-100× vs dense**, with the higher end if per-CVE popcount stays mostly absolute and the lower end if it scales linearly with fleet size.

The chart API hot path transfers the same bytes across the wire and AND-pop-counts them in Go, so wire and CPU savings move proportionally with storage.

## What Changes

### `encoding_type` column on `host_scd_data`

Add a `TINYINT NOT NULL DEFAULT 0` column that discriminates the format of `host_bitmap`:

- `encoding_type = 0` — **dense**: the existing bit-array layout, byte-for-byte unchanged.
- `encoding_type = 1` — **roaring**: standard `RoaringBitmap/roaring` portable serialization (`Bitmap.ToBytes()`).

The migration is `ALTER TABLE host_scd_data ADD COLUMN encoding_type TINYINT NOT NULL DEFAULT 0, ALGORITHM=INSTANT`. On MySQL 8.0+ (Fleet's floor: 8.0.44) this is a metadata-only operation, milliseconds regardless of row count. Existing rows logically read as `encoding_type = 0` (dense) by virtue of the column DEFAULT, correctly describing their format. No row data is rewritten by the schema migration itself.

### Type separation: storage form vs op form

`chart.Blob` (`Bytes []byte`, `Encoding uint8`) is the **storage form** — what reads from and writes to `host_scd_data`. `*roaring.Bitmap` is the **op form** — what every bitwise operation works on.

The two are joined by two boundary helpers:

- `chart.DecodeBitmap(Blob) (*roaring.Bitmap, error)` — converts storage → op form. Dispatches on `Encoding`; legacy dense blobs are walked once and inserted into a fresh roaring bitmap. Used by every DB read path.
- `chart.BitmapToBlob(*roaring.Bitmap) Blob` — converts op → storage form. Calls `RunOptimize()` before `ToBytes()`. Always emits `Encoding = EncodingRoaring`. Used by every DB write path.

This boundary discipline eliminates redundant decode work in op loops (decode once per row, op many times on the same `*roaring.Bitmap`), and confines all encoding awareness to the I/O edge — easier to simplify in a future cleanup change once dense is fully gone.

### Encoder

`chart.NewBitmap([]uint) *roaring.Bitmap` builds an in-memory bitmap from a host-id list, calling `RunOptimize()` before returning. `chart.HostIDsToBlob([]uint) chart.Blob` is a convenience that composes `NewBitmap` + `BitmapToBlob` for callers going directly from a host-id list to storage form.

The dense format remains valid for *reading* — pre-deploy rows are dense and remain so until converted by the in-place UPDATE (open rows) or aged out by retention (closed rows). After ~30 days, no production row is dense and `DecodeBitmap`'s dense branch can be removed in a follow-up change.

### Roaring-only operations

`chart.BlobAND`, `BlobOR`, `BlobANDNOT`, and `BlobPopcount` operate on `*roaring.Bitmap` operands and produce `*roaring.Bitmap` results (or `uint64` for popcount). They are thin wrappers over the corresponding `RoaringBitmap/roaring` library functions, kept as the Fleet-package namespace for stability and possible future extension.

There is no dense input to these functions — the storage-form `Blob` is decoded at the DB boundary. The popcount fast-path for dense bytes is gone; all popcount goes through `GetCardinality()`.

Working memory for ops is proportional to result cardinality, not to `max_host_id`. At 100k-host scale this is a ~100× reduction in transient memory per op vs the original decode-to-dense plan.

### In-place dense → roaring conversion in `recordSnapshot`

`recordSnapshot` already reads every open row on each cron tick. After the SELECT, every row is decoded to `*roaring.Bitmap` via `DecodeBitmap`. Rows whose source encoding was dense are re-serialized via `BitmapToBlob` and written back via a batched `UPDATE host_scd_data SET host_bitmap = ?, encoding_type = 1 WHERE id = ? AND encoding_type = 0`. The `encoding_type = 0` filter makes the UPDATE idempotent — a second pass UPDATEs zero rows.

The conversion is an in-place re-encoding, **not** a close+insert. `valid_from` and `valid_to` are unchanged because the host set is semantically unchanged; only the encoding differs. This avoids polluting the audit trail with spurious state-transition rows.

Change detection on the now-uniformly-roaring in-memory bitmaps uses `roaring.Bitmap.Equals()` (semantic equality on the bitmap structure) rather than `bytes.Equal` on serialized output. This sidesteps any roaring serialization quirks and makes the comparison independent of the storage layer.

After one cron tick post-deploy, every open row is roaring. Closed dense rows age out within 30 days via the existing retention cron. Net convergence to all-roaring within ~30 days, with zero operator intervention.

### Roaring serialization determinism (recommended, not load-bearing)

Every code path that produces a roaring blob — `BitmapToBlob` and by composition `HostIDsToBlob` — SHALL call `RunOptimize()` before `ToBytes()`. This keeps writes byte-deterministic for a given set, which is useful for observability (querying for byte-equal bitmaps), and avoids spurious storage churn if anything ever compares serialized bytes directly. With the move to `roaring.Equals()` for in-memory change detection, byte determinism is no longer strictly load-bearing for correctness — but it's nearly free to maintain and worth keeping.

### Library

`github.com/RoaringBitmap/roaring` — MIT license, no cgo, mature (InfluxDB / ClickHouse / Lucene ecosystem). Standard 32-bit roaring is sufficient (host IDs are uint and bounded well below 2^31).

## Capabilities

### Modified Capabilities

- `chart-bitmap-storage` — adds the roaring encoding alongside the existing dense encoding, discriminated by a new `encoding_type` column. All `Blob*` operations transparently handle both formats and produce a tagged `Blob` result. The format choice is per-row, picked at write time based on serialized size.

## Impact

- **Storage**: ~5× reduction measured at small-fleet scale (all rows in one chunk → array containers). Projected ~20-100× at 100k-host scale. Worst case is roaring's bitmap-container path for a kernel-tier CVE affecting most of the fleet, which is bounded to ~dense size + tens of bytes of roaring header — sub-percent overhead in a vanishingly rare case.
- **API latency**: roughly proportional reduction in wire transfer; CPU and memory savings from direct-on-roaring ops (no dense scratch buffer allocation).
- **Schema change**: one `ALTER TABLE ... ADD COLUMN ... ALGORITHM=INSTANT`. Milliseconds on 8.0+, no row rewrite.
- **No API contract change**: chart endpoint shape and semantics are identical.
- **Dependency**: new import of `github.com/RoaringBitmap/roaring`.
- **Test coverage**: existing `Blob*` tests need to be extended for mixed-encoding interop while legacy dense rows are still in the table. Cross-encoding tests must exercise all three roaring container types (array / bitmap / run). A determinism test must assert byte-equal output for the same set built via different code paths.
- **Convergence visibility**: a `SELECT COUNT(*) WHERE encoding_type = 0` query (and a one-shot startup log) makes lazy-migration progress directly observable. Column can be indexed if it becomes a frequent read.

## Out of Scope

- **64-bit roaring**. Host IDs are uint and never close to 2^31.
- **Encoding negotiation by client**. The chart endpoint contract doesn't expose the encoding; clients receive aggregated points.
- **Other-dataset retrofits**. Uptime / policy / future datasets share the same `Blob*` functions and benefit automatically.
- **Explicit backfill of closed rows**. The in-place open-row conversion + 30-day retention cron is sufficient for full convergence within one retention window.
- **Removing the dense decoder helper and `encoding_type` column after convergence**. Worth doing eventually — once production fleets are confirmed fully converged, a follow-up change can delete the decoder, drop the column, and simplify reads to assume roaring. Out of scope here.
