/**
 * Represents query params used as list options by the Fleet API
 */
export interface IListOptions {
  page: number;
  per_page: number;
  order_key: string;
  order_direction: string;
}

export type IListSort = Pick<IListOptions, "order_key" | "order_direction">;

export type IListPagination = Pick<IListOptions, "page" | "per_page">;
