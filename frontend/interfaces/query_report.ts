export interface IQueryReportResultRow {
  host_id: number;
  host_name: string;
  last_fetched: string;
  columns: any; // {col:val, ...}
}

export interface IQueryReportMeta {
  has_next_results: boolean;
  has_previous_results: boolean;
}

// Query report
export interface IQueryReport {
  query_id: number;
  results: IQueryReportResultRow[];
  report_clipped: boolean;
  /** Total number of results matching the request, across all pages. */
  count: number;
  /** Only present when the request was paginated with `per_page`. */
  meta?: IQueryReportMeta;
}
