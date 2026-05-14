## MODIFIED Requirements

### Requirement: host_scd_data SHALL discriminate bitmap encoding via an encoding_type column

The `host_scd_data` table SHALL include an `encoding_type TINYINT NOT NULL DEFAULT 0` column that identifies the format of the `host_bitmap` blob:

- `encoding_type = 0` — **Dense**: bit at position `n` set iff host `n` is in the set; total length `(max_id_in_set / 8) + 1`. Existing rows pre-dating the schema change SHALL read as dense by virtue of the column's DEFAULT.
- `encoding_type = 1` — **Roaring**: the standard portable `RoaringBitmap/roaring` serialization (`Bitmap.ToBytes()` output, post-`RunOptimize`).

The schema migration SHALL use `ALGORITHM=INSTANT`. Existing dense rows SHALL NOT be rewritten by the schema migration itself; they remain readable byte-for-byte and the column DEFAULT supplies their encoding identity.

New writes via `BitmapToBlob` SHALL always emit roaring (`Encoding = EncodingRoaring`). The dense write path is removed; the dense format remains a supported READ format for pre-deploy rows until they are converted in place (open rows) or aged out by retention (closed rows).

Empty bitmaps SHALL serialize as `Blob{Bytes: nil, Encoding: EncodingRoaring}`. The encoding tag is semantically irrelevant for nil bytes but tagging as Roaring keeps the "all new writes are roaring" invariant honest.

#### Scenario: A sparse 1-host bitmap is stored as roaring

- **GIVEN** a CVE affects exactly host id 7
- **WHEN** `HostIDsToBlob([7])` is called
- **THEN** the returned blob SHALL have `Encoding = EncodingRoaring` (column value 1)
- **AND** `DecodeBitmap` on the blob SHALL produce a bitmap whose set bits equal `{7}`

#### Scenario: A dense-random middle-density bitmap is still stored as roaring

- **GIVEN** a CVE affects ~50% of hosts randomly distributed across the host id range
- **WHEN** `HostIDsToBlob(ids)` is called
- **THEN** the returned blob SHALL have `Encoding = EncodingRoaring`
- **AND** `DecodeBitmap` on the blob SHALL produce a bitmap whose set bits equal the input host set
- **AND** the byte length MAY exceed what a dense encoding would have produced (acceptable; the worst-case overhead is sub-percent and confined to a narrow band of bitmap shapes)

#### Scenario: Empty input produces a nil roaring blob

- **GIVEN** an empty `[]uint`
- **WHEN** `HostIDsToBlob([])` is called
- **THEN** the returned blob SHALL have `Bytes = nil` and `Encoding = EncodingRoaring`

#### Scenario: An existing dense row is preserved by the schema migration

- **GIVEN** a `host_scd_data` row written before this change, with `host_bitmap` containing the dense byte representation of some host set
- **WHEN** the `ALTER TABLE ... ADD COLUMN encoding_type TINYINT NOT NULL DEFAULT 0, ALGORITHM=INSTANT` migration completes
- **THEN** the row's `host_bitmap` bytes SHALL be byte-for-byte unchanged (INSTANT ALTER does not rewrite row data)
- **AND** the row's `encoding_type` SHALL read back as 0 by virtue of the column's DEFAULT

#### Scenario: Post-change reads of a legacy dense row produce semantically correct results

- **GIVEN** a `host_scd_data` row with `encoding_type = 0` (legacy, written before the migration) whose `host_bitmap` is the dense byte representation of host set `{3, 7, 11}`
- **WHEN** the row is read and passed to `DecodeBitmap`
- **THEN** the resulting `*roaring.Bitmap` SHALL have `GetCardinality()` equal to `3`
- **AND** ANDing it with a bitmap representing `{7, 99}` SHALL produce a bitmap equal to `{7}`
- **AND** ORing it with a bitmap representing `{99}` SHALL produce a bitmap equal to `{3, 7, 11, 99}`
- **AND** AndNot'ing it with a bitmap representing `{7}` SHALL produce a bitmap equal to `{3, 11}`

### Requirement: Roaring serialization SHALL be deterministic

`BitmapToBlob` SHALL call `roaring.Bitmap.RunOptimize()` before `ToBytes()`. By composition, every code path that serializes a roaring bitmap for storage (e.g. `HostIDsToBlob`) SHALL call `RunOptimize` exactly once before serialization.

Determinism is not load-bearing for correctness of in-memory change detection (`recordSnapshot` uses `roaring.Bitmap.Equals()` on decoded bitmaps, not `bytes.Equal` on serialized output). However, determinism is preserved as a desirable storage property: deterministic writes simplify observability tooling (byte-level duplicate detection), avoid spurious churn from future code paths that may compare bytes, and make reproducing serialized blobs trivial.

#### Scenario: The same host set built via different code paths produces byte-equal serialized blobs

- **GIVEN** host set `{2, 100, 65540, 130000}`
- **WHEN** the set is serialized via `BitmapToBlob(NewBitmap(ids))`, AND via `BitmapToBlob(DecodeBitmap(denseBlob))` of the same set, AND via `BitmapToBlob(BlobOR(empty, NewBitmap(ids)))`
- **THEN** all three resulting `Blob.Bytes` slices SHALL be byte-equal

### Requirement: Storage form and op form SHALL be separated

The chart package SHALL define two distinct types:

- `chart.Blob{Bytes []byte, Encoding uint8}` — storage form. Used only at the database I/O boundary.
- `*roaring.Bitmap` — op form. Used for all bitwise operations and in-memory representation.

The bitwise operations `BlobAND`, `BlobOR`, `BlobANDNOT`, and `BlobPopcount` SHALL operate on `*roaring.Bitmap` operands and return `*roaring.Bitmap` (or `uint64` for popcount). They SHALL NOT accept `chart.Blob` arguments — encoding-awareness is confined to `DecodeBitmap` and `BitmapToBlob` at the I/O edge.

Callers reading rows from `host_scd_data` SHALL pass each row's `(host_bitmap, encoding_type)` through `DecodeBitmap` to obtain a `*roaring.Bitmap` before invoking any op. Callers writing rows SHALL serialize their final `*roaring.Bitmap` via `BitmapToBlob` and persist both the `host_bitmap` bytes and the resulting `encoding_type`.

`BlobPopcount` SHALL be implemented as `b.GetCardinality()`. There SHALL NOT be a special-case dense fast-path at the op layer — dense inputs do not reach ops.

#### Scenario: A legacy dense row is decoded once at the boundary before ops

- **GIVEN** an open `host_scd_data` row with `encoding_type = 0` representing host set `{1, 5, 9}` and an in-memory `*roaring.Bitmap` `b` representing `{5, 9, 15}`
- **WHEN** code wants to compute the intersection
- **THEN** the row SHALL be passed to `DecodeBitmap` to produce a `*roaring.Bitmap` `a`
- **AND** `BlobAND(a, b)` SHALL be called on the two roaring bitmaps
- **AND** the result SHALL be a `*roaring.Bitmap` equal to `{5, 9}`

#### Scenario: Accumulation loop does not re-decode the accumulator

- **GIVEN** a chart-API query that ORs together the bitmaps of N rows fetched from the database
- **WHEN** the loop runs
- **THEN** the accumulator SHALL be held as a single `*roaring.Bitmap` for the duration of the loop
- **AND** each row's bitmap SHALL be decoded exactly once (at iteration entry) via `DecodeBitmap`
- **AND** the accumulator SHALL NOT be re-decoded on each iteration

### Requirement: Snapshot write path SHALL convert dense open rows to roaring in place

`recordSnapshot` in the chart datastore SHALL, on each cron tick:

1. Read open rows including `(id, entity_id, host_bitmap, encoding_type, valid_from)`.
2. Decode each row to `*roaring.Bitmap` via `DecodeBitmap`.
3. For any row whose source `encoding_type = 0`: serialize the just-decoded bitmap via `BitmapToBlob` and issue a batched `UPDATE host_scd_data SET host_bitmap = ?, encoding_type = 1 WHERE id = ? AND encoding_type = 0`. The `encoding_type = 0` filter clause SHALL make the UPDATE idempotent under stale reads (a second pass UPDATEs zero rows).
4. Detect changes by comparing the decoded existing bitmap to the incoming bitmap via `roaring.Bitmap.Equals()`. Skip writes for unchanged entities.
5. For changed entities: close existing rows via UPDATE valid_to, insert new rows by serializing the incoming `*roaring.Bitmap` via `BitmapToBlob`.

The in-place UPDATE in step 3 SHALL NOT change `valid_from` or `valid_to` and SHALL NOT produce a close+insert pair. The host set is semantically unchanged; only the encoding changes. This preserves audit-trail truthfulness — re-encoding is not a state transition.

Change detection SHALL use `roaring.Bitmap.Equals()` on decoded bitmaps rather than `bytes.Equal` on serialized output. This makes the comparison independent of serialization determinism and avoids redundant serialization just to compare.

#### Scenario: Open dense row is converted in place on first cron tick

- **GIVEN** an open row in `host_scd_data` for entity `CVE-X` with `encoding_type = 0` and a dense `host_bitmap` representing host set `{3, 7, 11}`
- **WHEN** the next cron tick computes a new bitmap for `CVE-X` containing the same host set `{3, 7, 11}`
- **THEN** the existing row's `host_bitmap` SHALL be UPDATEd in place to its roaring representation
- **AND** the existing row's `encoding_type` SHALL be UPDATEd to 1
- **AND** the existing row's `valid_from` and `valid_to` SHALL be unchanged
- **AND** no new row SHALL be inserted
- **AND** the row SHALL NOT be closed (no `valid_to` set on the existing open row)

#### Scenario: Subsequent cron tick on a roaring row with unchanged hosts is a no-op

- **GIVEN** an open row for entity `CVE-X` with `encoding_type = 1` (already converted)
- **WHEN** the next cron tick computes a new bitmap for `CVE-X` containing the same host set
- **THEN** the in-place UPDATE statement SHALL match zero rows (the `encoding_type = 0` filter is false)
- **AND** the `roaring.Equals()` change-detection SHALL skip the row
- **AND** no writes SHALL be issued for the row

#### Scenario: Open dense row whose host set changed gets converted then close-and-insert

- **GIVEN** an open row for entity `CVE-X` with `encoding_type = 0` representing `{3, 7, 11}`
- **WHEN** the next cron tick computes a new bitmap for `CVE-X` containing `{3, 7, 11, 99}`
- **THEN** the in-place UPDATE first converts the row to roaring (encoding_type = 1, same host set, valid_from / valid_to unchanged)
- **AND** the subsequent `roaring.Equals()` check against the new bitmap returns false
- **AND** the row is then closed with `valid_to` set to the new `bucketStart`
- **AND** a new open row is inserted for the new host set with `encoding_type = 1` (roaring)
