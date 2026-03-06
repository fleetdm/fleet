import { getGitOpsModeTipContent } from "utilities/helpers";

import {
  buildActionOptions,
  ACTION_EDIT_APPEARANCE,
  ACTION_EDIT_SOFTWARE,
  ACTION_EDIT_CONFIGURATION,
  ACTION_PATCH,
  ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
} from "./SoftwareDetailsSummary";

describe("buildActionOptions", () => {
  it("returns Edit appearance and Edit software when non-Android, non-gitops, no extra permissions", () => {
    const result = buildActionOptions(
      false, // gitOpsModeEnabled
      undefined,
      undefined,
      false, // androidSoftwareAvailableForInstall
      false, // canAddPatchPolicy
      false, // canConfigureAutoUpdate
      false // hasExistingPatchPolicy
    );

    expect(result).toEqual([
      {
        label: "Edit appearance",
        value: ACTION_EDIT_APPEARANCE,
        isDisabled: false,
        tooltipContent: undefined,
      },
      {
        label: "Edit software",
        value: ACTION_EDIT_SOFTWARE,
        isDisabled: false,
        tooltipContent: undefined,
      },
    ]);
  });

  it("hides Edit software and shows Edit configuration for Android installers", () => {
    const result = buildActionOptions(
      false,
      undefined,
      undefined,
      true, // androidSoftwareAvailableForInstall
      false,
      false,
      false
    );

    const values = result.map((o) => o.value);

    expect(values).toContain(ACTION_EDIT_CONFIGURATION);
    expect(values).not.toContain(ACTION_EDIT_SOFTWARE);

    const editConfig = result.find(
      (opt) => opt.value === ACTION_EDIT_CONFIGURATION
    );
    expect(editConfig).toEqual({
      label: "Edit configuration",
      value: ACTION_EDIT_CONFIGURATION,
      isDisabled: false,
      tooltipContent: undefined,
    });
  });

  it("applies gitops tooltip to Edit appearance and Edit configuration, and to Edit software for vpp_apps", () => {
    const result = buildActionOptions(
      true, // gitOpsModeEnabled
      "https://repo.git", // repoURL
      "vpp_apps", // source
      true, // androidSoftwareAvailableForInstall
      false,
      false,
      false
    );

    const editAppearance = result.find(
      (opt) => opt.value === ACTION_EDIT_APPEARANCE
    );
    const editConfig = result.find(
      (opt) => opt.value === ACTION_EDIT_CONFIGURATION
    );
    const editSoftware = result.find(
      (opt) => opt.value === ACTION_EDIT_SOFTWARE
    );

    expect(editAppearance).toMatchObject({
      isDisabled: true,
      tooltipContent: /manage in/i,
    });

    expect(editConfig).toMatchObject({
      isDisabled: true,
      tooltipContent: /manage in/,
    });

    // For vpp_apps and non-Android, Edit software would also get the tooltip.
    expect(editSoftware).toBeUndefined();
  });

  it("adds Patch option enabled when canAddPatchPolicy and no existing patch policy", () => {
    const result = buildActionOptions(
      false,
      undefined,
      undefined,
      false,
      true, // canAddPatchPolicy
      false, // canConfigureAutoUpdate
      false // hasExistingPatchPolicy
    );

    const patch = result.find((opt) => opt.value === ACTION_PATCH);

    expect(patch).toEqual({
      label: "Patch",
      value: ACTION_PATCH,
      isDisabled: false,
      tooltipContent: undefined,
    });
  });

  it("adds Patch option disabled with tooltip when hasExistingPatchPolicy", () => {
    const result = buildActionOptions(
      false,
      undefined,
      undefined,
      false,
      true, // canAddPatchPolicy
      false, // canConfigureAutoUpdate
      true // hasExistingPatchPolicy
    );

    const patch = result.find((opt) => opt.value === ACTION_PATCH);

    expect(patch).toEqual({
      label: "Patch",
      value: ACTION_PATCH,
      isDisabled: true,
      tooltipContent: "Patch policy is already added.",
    });
  });

  it("adds Schedule auto updates option when canConfigureAutoUpdate", () => {
    const result = buildActionOptions(
      false,
      undefined,
      undefined,
      false,
      false,
      true, // canConfigureAutoUpdate
      false
    );

    const autoUpdate = result.find(
      (opt) => opt.value === ACTION_EDIT_AUTO_UPDATE_CONFIGURATION
    );

    expect(autoUpdate).toEqual({
      label: "Schedule auto updates",
      value: ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
    });
  });
});
