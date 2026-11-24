import {
  getHostDeviceStatusUIState,
  getWipeRequestedUIState,
  isDeviceStatusUpdating,
} from "./helpers";

describe("getHostDeviceStatusUIState", () => {
  it.each([
    { deviceStatus: "unlocked", pendingAction: "", want: "unlocked" },
    { deviceStatus: "unlocked", pendingAction: "wipe", want: "wiping" },
    { deviceStatus: "unlocked", pendingAction: "reset", want: "resetting" },
    { deviceStatus: "reset", pendingAction: "", want: "reset" },
    {
      deviceStatus: "reset",
      pendingAction: "wipe",
      want: "reset_wipe_pending",
    },
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

describe("isDeviceStatusUpdating", () => {
  it.each(["unenrolling", "resetting"] as const)(
    "treats %s as updating",
    (state) => {
      expect(isDeviceStatusUpdating(state)).toBe(true);
    }
  );
});

describe("getWipeRequestedUIState", () => {
  it.each([
    { current: "unlocked", want: "wiping" },
    { current: "resetting", want: "wiping" },
    { current: "reset", want: "reset_wipe_pending" },
  ] as const)("goes from $current to $want", ({ current, want }) => {
    expect(getWipeRequestedUIState(current)).toBe(want);
  });
});
