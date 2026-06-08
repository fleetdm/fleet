// Note: We intentionally test getOriginalSoftwareState, getCurrentSoftwareState,
// and getTrulyDirtyInstallSoftwareItems in isolation instead of writing a
// full integration test for InstallSoftwareModal’s onSubmit. A realistic
// integration test would need standing up PoliciesPaginatedList, pagination,
// and react-query data loading, which adds a lot of brittle setup for minor
// extra confidence over these focused unit tests.

import {
  getOriginalSoftwareState,
  getCurrentSoftwareState,
  getTrulyDirtyInstallSoftwareItems,
} from "./InstallSoftwareModal";
import { IFormPolicy } from "../PoliciesPaginatedList/PoliciesPaginatedList";

const MOCK_POLICY_SOFTWARE = {
  software_title_id: 20,
  name: "1Password",
  display_name: "",
};

const createMockPolicyForInstallSoftware = (
  overrides: Partial<IFormPolicy> = {}
): IFormPolicy =>
  ({
    id: 1,
    name: "Policy",
    platform: "darwin",
    installSoftwareEnabled: false,
    swIdToInstall: undefined,
    ...overrides,
  } as IFormPolicy);

describe("getOriginalSoftwareState", () => {
  it("extracts original software id when present", () => {
    const policy = createMockPolicyForInstallSoftware({
      install_software: MOCK_POLICY_SOFTWARE,
    });
    const { originallyEnabled, originalSwId } = getOriginalSoftwareState(
      policy
    );

    expect(originallyEnabled).toBe(true);
    expect(originalSwId).toBe(20);
  });
});

describe("getCurrentSoftwareState", () => {
  it("normalizes disabled state and null-ish swIdToInstall", () => {
    const policy = createMockPolicyForInstallSoftware({
      installSoftwareEnabled: false,
      swIdToInstall: undefined,
    });

    const { nowEnabled, nowSwId } = getCurrentSoftwareState(policy);

    expect(nowEnabled).toBe(false);
    expect(nowSwId).toBeNull();
  });

  it("returns enabled with selected software id", () => {
    const policy = createMockPolicyForInstallSoftware({
      installSoftwareEnabled: true,
      swIdToInstall: 99,
    });

    const { nowEnabled, nowSwId } = getCurrentSoftwareState(policy);

    expect(nowEnabled).toBe(true);
    expect(nowSwId).toBe(99);
  });
});

describe("getTrulyDirtyInstallSoftwareItems", () => {
  it("returns only policies that changed enablement or software id", () => {
    const dirtyItems: IFormPolicy[] = [
      // 1. Unchanged: originally enabled, still enabled, same software -> excluded
      createMockPolicyForInstallSoftware({
        id: 1,
        install_software: MOCK_POLICY_SOFTWARE,
        installSoftwareEnabled: true,
        swIdToInstall: MOCK_POLICY_SOFTWARE.software_title_id,
      }),

      // 2. Turned on: originally disabled, now enabled with swId -> included
      createMockPolicyForInstallSoftware({
        id: 2,
        installSoftwareEnabled: true,
        swIdToInstall: 20,
      }),

      // 3. Turned off: originally enabled, now disabled -> included
      createMockPolicyForInstallSoftware({
        id: 3,
        install_software: MOCK_POLICY_SOFTWARE,
        installSoftwareEnabled: false,
        swIdToInstall: undefined,
      }),

      // 4. Software changed: originally enabled with A, now enabled with B -> included
      createMockPolicyForInstallSoftware({
        id: 4,
        install_software: MOCK_POLICY_SOFTWARE,
        installSoftwareEnabled: true,
        swIdToInstall: 30,
      }),
    ];

    const result = getTrulyDirtyInstallSoftwareItems(dirtyItems);
    const ids = result.map((p) => p.id).sort();

    expect(ids).toEqual([2, 3, 4]);
  });
});
