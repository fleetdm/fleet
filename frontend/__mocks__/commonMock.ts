import {
  ListEntitiesResponseCommon,
  ListEntitiesResponsePaginationCommon,
} from "services/entities/common";

const DEFAULT_PAGINATION_RESPONSE: ListEntitiesResponsePaginationCommon = {
  has_next_results: false,
  has_previous_results: false,
};

export const createMockPaginationResponse = (
  overrides?: Partial<ListEntitiesResponsePaginationCommon>
): typeof DEFAULT_PAGINATION_RESPONSE => {
  return { ...DEFAULT_PAGINATION_RESPONSE, ...overrides };
};
const DEFAULT_LIST_ENTITIES_RESPONSE_COMMON_MOCK: ListEntitiesResponseCommon = {
  meta: createMockPaginationResponse(),
  count: 1,
};

export const createMockListEntitiesResponseCommon = (
  overrides?: Partial<ListEntitiesResponseCommon>
): ListEntitiesResponseCommon => {
  return { ...DEFAULT_LIST_ENTITIES_RESPONSE_COMMON_MOCK, ...overrides };
};
