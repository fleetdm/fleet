// TODO - apply broadly
interface PaginationMeta {
  has_next_results: boolean;
  has_previous_results: boolean;
}

export interface ListEntitiesMeta {
  meta: PaginationMeta;
  count: number;
}
