import sendRequest from "services";

import teamPoliciesAPI from "./team_policies";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

describe("teamPoliciesAPI patch policy flags", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({});
  });

  it("forwards both flags when creating a patch policy", async () => {
    await teamPoliciesAPI.create({
      team_id: 1,
      type: "patch",
      patch_software_title_id: 10,
      patch_when_closed: true,
      continuous_automations_enabled: true,
    });

    expect(mockSendRequest).toHaveBeenCalledWith(
      "POST",
      expect.stringContaining("/1/policies"),
      expect.objectContaining({
        patch_when_closed: true,
        continuous_automations_enabled: true,
      })
    );
  });

  it("retains false flag values when updating a patch policy", async () => {
    await teamPoliciesAPI.update(22, {
      team_id: 1,
      patch_when_closed: false,
      continuous_automations_enabled: false,
    });

    expect(mockSendRequest).toHaveBeenCalledWith(
      "PATCH",
      expect.stringContaining("/1/policies/22"),
      expect.objectContaining({
        patch_when_closed: false,
        continuous_automations_enabled: false,
      })
    );
  });
});
