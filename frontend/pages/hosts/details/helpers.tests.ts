import { getHostDeviceStatusUIState } from "./helpers";

describe("getHostDeviceStatusUIState", () => {
  it.each([
    { deviceStatus: "unlocked", pendingAction: "", want: "unlocked" },
    { deviceStatus: "unlocked", pendingAction: "wipe", want: "wiping" },
    { deviceStatus: "unlocked", pendingAction: "location", want: "locking" },
    { deviceStatus: "locked", pendingAction: "location", want: "locating" },
    {
      deviceStatus: "unlocked",
      pendingAction: "unenroll",
      want: "unenrolling",
    },
  ] as const)(
    "maps $deviceStatus + '$pendingAction' to $want",
    ({ deviceStatus, pendingAction, want }) => {
      expect(getHostDeviceStatusUIState(deviceStatus, pendingAction)).toBe(
        want
      );
    }
  );
});
