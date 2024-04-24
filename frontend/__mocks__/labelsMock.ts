import { ILabel } from "interfaces/label";
import { IGetLabelResonse } from "services/entities/labels";

const DEFAULT_LABEL_MOCK: ILabel = {
  created_at: "2024-04-12T13:32:00Z",
  updated_at: "2024-04-12T14:27:07Z",
  id: 1,
  name: "test label",
  description: "test label description",
  query: "SELECT 1;",
  platform: "darwin",
  label_type: "regular",
  label_membership_type: "dynamic",
  display_text: "test macsss",
  count: 0,
  host_ids: null,
};

export const createMockLabel = (overrides?: Partial<ILabel>): ILabel => {
  return { ...DEFAULT_LABEL_MOCK, ...overrides };
};

const DEFAULT_GET_LABEL_RESPONSE_MOCK: IGetLabelResonse = {
  label: createMockLabel(),
};

export const createMockGetLabelResponse = (
  overrides?: Partial<IGetLabelResonse>
): IGetLabelResonse => {
  return { ...DEFAULT_GET_LABEL_RESPONSE_MOCK, ...overrides };
};
