export interface IQueryReportResultRow {
  host_id: number;
  host_name: string;
  last_fetched: string;
  columns: Record<string, string>;
}

// Query report
export interface IQueryReport {
  query_id: number;
  results: IQueryReportResultRow[];
}
