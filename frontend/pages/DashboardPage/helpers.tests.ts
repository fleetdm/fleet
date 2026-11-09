import { getPlatformHostCounts } from "./helpers";

describe("getPlatformHostCounts", () => {
  it("adds every Linux distribution to the Linux total", () => {
    expect(
      getPlatformHostCounts([
        { platform: "linux", hosts_count: 1 },
        { platform: "debian", hosts_count: 2 },
        { platform: "ubuntu", hosts_count: 4 },
        { platform: "darwin", hosts_count: 5 },
      ])
    ).toEqual({
      darwin: 5,
      windows: 0,
      linux: 7,
      chrome: 0,
      ios: 0,
      ipados: 0,
      android: 0,
    });
  });

  it("returns zeros without platforms", () => {
    expect(getPlatformHostCounts(null).linux).toBe(0);
  });
});
