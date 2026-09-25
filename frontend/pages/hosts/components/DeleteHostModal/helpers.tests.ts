import createMockHost from "__mocks__/hostMock";
import { IHost } from "interfaces/host";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import { HostPlatform } from "interfaces/platform";

import { getSharedDeleteHostTarget } from "./helpers";

const host = (
  id: number,
  platform: HostPlatform,
  connectedToFleet = false,
  enrollmentStatus: MdmEnrollmentStatus | null = null
): IHost => {
  const mock = createMockHost({ id, platform });
  mock.mdm = {
    ...mock.mdm,
    connected_to_fleet: connectedToFleet,
    enrollment_status: enrollmentStatus,
  };
  return mock;
};

describe("getSharedDeleteHostTarget", () => {
  it("returns nothing for an empty selection", () => {
    expect(getSharedDeleteHostTarget([])).toBeUndefined();
  });

  it("returns the single host's platform and MDM state", () => {
    expect(
      getSharedDeleteHostTarget([host(1, "darwin", true, "On (automatic)")])
    ).toEqual({
      platform: "darwin",
      isMdmEnrolledInFleet: true,
      mdmEnrollmentStatus: "On (automatic)",
    });
  });

  it("treats Linux distributions as one group", () => {
    expect(
      getSharedDeleteHostTarget([host(1, "ubuntu"), host(2, "debian")])
    ).toMatchObject({ platform: "ubuntu" });
  });

  it("treats iOS and iPadOS as one group", () => {
    expect(
      getSharedDeleteHostTarget([host(1, "ios"), host(2, "ipados")])
    ).toMatchObject({ platform: "ios" });
  });

  it("ignores MDM state differences for platforms whose copy does not depend on it", () => {
    expect(
      getSharedDeleteHostTarget([
        host(1, "windows", true, "On (automatic)"),
        host(2, "windows", false, null),
      ])
    ).toMatchObject({ platform: "windows" });
  });

  it("requires the same MDM state for macOS hosts", () => {
    expect(
      getSharedDeleteHostTarget([
        host(1, "darwin", true, "On (automatic)"),
        host(2, "darwin", true, "On (manual)"),
      ])
    ).toBeUndefined();
    expect(
      getSharedDeleteHostTarget([
        host(1, "darwin", true, "On (manual)"),
        host(2, "darwin", false, null),
      ])
    ).toBeUndefined();
  });

  it("returns nothing for a mixed-platform selection", () => {
    expect(
      getSharedDeleteHostTarget([host(1, "darwin"), host(2, "windows")])
    ).toBeUndefined();
  });

  it("returns nothing for platforms without per-platform copy", () => {
    expect(getSharedDeleteHostTarget([host(1, "chrome")])).toBeUndefined();
  });
});
