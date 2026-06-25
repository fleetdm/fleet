# Chart bounded context

Before working in `server/chart/`, read [README.md](./README.md) — it covers the
architecture, the SCD + bitmap data model, sample strategies, and a step-by-step
guide for adding new datasets and charts.

Quick reminders (full detail in the README):

- This is a **self-contained bounded context**. No chart package may import
  `server/fleet` or `server/contexts/viewer` — `arch_test.go` enforces this. Bridge
  to legacy Fleet via a narrow `api` interface implemented in `server/acl/chartacl`.
- External code only touches `bootstrap.New` and the `api` package.
- Data lives in one table, `host_scd_data` (slowly-changing-dimension type-2), with
  host-sets stored as roaring bitmaps. Work in op form (`*roaring.Bitmap`); only
  cross the storage boundary via `BitmapToBlob` / `DecodeBitmap`.
- Mind the **nil vs empty-slice** semantics on `HostFilter.TeamIDs` and
  `GetSCDData` `entityIDs` — they are load-bearing; don't normalize them away.
- Snapshot collectors must call `RecordBucketData` even with an empty map (it closes
  open rows). Accumulate collectors may skip empty input.
- Adding a dataset: implement `api.Dataset` in `datasets.go`, add any store method to
  **both** `api.DatasetStore` and `internal/types.Datastore`, register it in
  `cmd/fleet/serve.go`, and wire config gating + scrub in
  `server/fleet/historical_data.go` if it's opt-in/out.
