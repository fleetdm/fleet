import { ILabel } from "interfaces/label";

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
  fleet_id: null,
  criteria: {
    vital: "end_user_idp_department",
    value: " IT admins",
  },
};

// eslint-disable-next-line import/prefer-default-export
export const createMockLabel = (overrides?: Partial<ILabel>): ILabel => {
  return { ...DEFAULT_LABEL_MOCK, ...overrides };
};

