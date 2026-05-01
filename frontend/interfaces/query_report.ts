export interface IQueryReportResultRow {
  host_id: number;
  host_name: string;
  last_fetched: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  columns: any; // {col:val, ...}
}

// Query report
export interface IQueryReport {
  query_id: number;
  results: IQueryReportResultRow[];
  report_clipped: boolean;
}
