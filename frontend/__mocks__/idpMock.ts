import { IGetSCIMDetailsResponse } from "services/entities/idp";

const DEFAULT_GET_SCIM_DETAILS_MOCK: IGetSCIMDetailsResponse = {
  last_request: {
    requested_at: "2025-01-29T17:00:00Z",
    status: "success",
    details: "Successfully received end user information from your IdP",
  },
};

// eslint-disable-next-line import/prefer-default-export
export const createMockGetSCIMDetailsResponse = (
  overrides?: Partial<IGetSCIMDetailsResponse>
): IGetSCIMDetailsResponse => {
  return { ...DEFAULT_GET_SCIM_DETAILS_MOCK, ...overrides };
};
