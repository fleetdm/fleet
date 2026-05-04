import {
  buildActionOptions,
  ACTION_EDIT_APPEARANCE,
  ACTION_EDIT_SOFTWARE,
  ACTION_EDIT_CONFIGURATION,
  ACTION_PATCH,
  ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
} from "./SoftwareDetailsSummary";

describe("buildActionOptions", () => {
  it("returns only Edit appearance when user cannot edit software or configuration and cannot patch or configure auto updates", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canAddPatchPolicy: false,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: false,
    });

    expect(result).toEqual([
      {
        label: "Edit appearance",
        value: ACTION_EDIT_APPEARANCE,
        isDisabled: false,
        tooltipContent: undefined,
      },
    ]);
  });

  it("adds Edit software when canEditSoftware", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: true,
      canEditConfiguration: false,
      canAddPatchPolicy: false,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: false,
    });

    const values = result.map((o) => o.value);
    expect(values).toContain(ACTION_EDIT_SOFTWARE);

    const editSoftware = result.find(
      (opt) => opt.value === ACTION_EDIT_SOFTWARE
    );
    expect(editSoftware).toEqual({
      label: "Edit software",
      value: ACTION_EDIT_SOFTWARE,
      isDisabled: false,
      tooltipContent: undefined,
    });
  });

  it("adds Edit configuration when canEditConfiguration", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: false,
      canEditConfiguration: true,
      canAddPatchPolicy: false,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: false,
    });

    const values = result.map((o) => o.value);
    expect(values).toContain(ACTION_EDIT_CONFIGURATION);

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
    const result = buildActionOptions({
      gitOpsModeEnabled: true,
      repoURL: "https://repo.git",
      source: "vpp_apps",
      canEditSoftware: true,
      canEditConfiguration: true,
      canAddPatchPolicy: false,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: false,
    });

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
      tooltipContent: expect.anything(),
    });

    expect(editConfig).toMatchObject({
      isDisabled: true,
      tooltipContent: expect.anything(),
    });

    // For vpp_apps, Edit software also gets the gitops tooltip if present.
    expect(editSoftware).toMatchObject({
      isDisabled: true,
      tooltipContent: expect.anything(),
    });
  });

  it("adds Patch option enabled when canAddPatchPolicy and no existing patch policy", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canAddPatchPolicy: true,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: false,
    });

    const patch = result.find((opt) => opt.value === ACTION_PATCH);

    expect(patch).toEqual({
      label: "Patch",
      value: ACTION_PATCH,
      isDisabled: false,
      tooltipContent: undefined,
    });
  });

  it("adds Patch option disabled with tooltip when hasExistingPatchPolicy", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canAddPatchPolicy: true,
      canConfigureAutoUpdate: false,
      hasExistingPatchPolicy: true,
    });

    const patch = result.find((opt) => opt.value === ACTION_PATCH);

    expect(patch).toEqual({
      label: "Patch",
      value: ACTION_PATCH,
      isDisabled: true,
      tooltipContent: "Patch policy is already added.",
    });
  });

  it("adds Schedule auto updates option when canConfigureAutoUpdate", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      source: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canAddPatchPolicy: false,
      canConfigureAutoUpdate: true,
      hasExistingPatchPolicy: false,
    });

    const autoUpdate = result.find(
      (opt) => opt.value === ACTION_EDIT_AUTO_UPDATE_CONFIGURATION
    );

    expect(autoUpdate).toEqual({
      label: "Schedule auto updates",
      value: ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
    });
  });
});
