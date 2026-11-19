import sendRequest from "services";

import hostCountAPI from "./host_count";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

describe("hostCountAPI.load", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({ count: 0 });
  });

  it("sends the AB issues filter alongside a label", async () => {
    await hostCountAPI.load({
      selectedLabels: ["labels/7"],
      depProfileError: true,
    });

    const path = mockSendRequest.mock.calls[0][1];
    expect(path).toContain("label_id=7");
    expect(path).toContain("dep_profile_error=true");
  });

  it("sends the AB profile assignment filter", async () => {
    await hostCountAPI.load({ depAssignProfileResponse: "FAILED" });

    expect(mockSendRequest.mock.calls[0][1]).toContain(
      "dep_assign_profile_response=FAILED"
    );
  });
});
