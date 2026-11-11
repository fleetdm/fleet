import sendRequest from "services";

import mdmAndroidAPI from "./mdm_android";

jest.mock("services", () => ({
  __esModule: true,
  default: jest.fn(),
}));

const mockSendRequest = sendRequest as jest.MockedFunction<typeof sendRequest>;

const ZERO_TOUCH_PATH =
  "/latest/fleet/android_enterprise/zero_touch_configuration";

describe("mdmAndroidAPI.getZeroTouchConfiguration", () => {
  beforeEach(() => {
    mockSendRequest.mockReset();
    mockSendRequest.mockResolvedValue({});
  });

  it.each([
    ["no fleet", undefined, ZERO_TOUCH_PATH],
    ["Unassigned (0)", 0, ZERO_TOUCH_PATH],
    ["a fleet", 7, `${ZERO_TOUCH_PATH}?fleet_id=7`],
  ])("requests %s", async (_, fleetId, expectedPath) => {
    await mdmAndroidAPI.getZeroTouchConfiguration(fleetId);
    expect(mockSendRequest).toHaveBeenCalledWith("GET", expectedPath);
  });
});
