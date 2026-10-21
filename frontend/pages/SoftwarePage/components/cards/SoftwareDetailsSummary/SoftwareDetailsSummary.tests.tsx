import { render, screen } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import SoftwareDetailsSummary, {
  buildActionOptions,
  ACTION_EDIT_APPEARANCE,
  ACTION_EDIT_SOFTWARE,
  ACTION_EDIT_CONFIGURATION,
  ACTION_DEPLOY,
  ACTION_VERSIONS,
  ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
} from "./SoftwareDetailsSummary";

// SoftwareIcon calls an API via useQuery; stub it out for these unit tests.
jest.mock("../../icons/SoftwareIcon", () => ({
  __esModule: true,
  default: () => <div data-testid="software-icon" />,
}));

describe("buildActionOptions", () => {
  it("returns only Edit appearance when user cannot edit software or configuration and cannot patch or configure auto updates", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
    });

    expect(result).toEqual([
      {
        label: "Edit appearance",
        value: ACTION_EDIT_APPEARANCE,
        disabled: false,
        tooltipContent: undefined,
      },
    ]);
  });

  it("adds Edit software when canEditSoftware", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: true,
      canEditConfiguration: false,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
    });

    const values = result.map((o) => o.value);
    expect(values).toContain(ACTION_EDIT_SOFTWARE);

    const editSoftware = result.find(
      (opt) => opt.value === ACTION_EDIT_SOFTWARE
    );
    expect(editSoftware).toEqual({
      label: "Edit software",
      value: ACTION_EDIT_SOFTWARE,
      disabled: false,
      tooltipContent: undefined,
    });
  });

  it("adds Edit configuration when canEditConfiguration", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: false,
      canEditConfiguration: true,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
    });

    const values = result.map((o) => o.value);
    expect(values).toContain(ACTION_EDIT_CONFIGURATION);

    const editConfig = result.find(
      (opt) => opt.value === ACTION_EDIT_CONFIGURATION
    );
    expect(editConfig).toEqual({
      label: "Edit configuration",
      value: ACTION_EDIT_CONFIGURATION,
      disabled: false,
      tooltipContent: undefined,
    });
  });

  it("applies gitops tooltip to Edit appearance and Edit configuration, and to Edit software for Apple VPP", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: true,
      repoURL: "https://repo.git",
      isAppleVpp: true,
      canEditSoftware: true,
      canEditConfiguration: true,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
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
      disabled: true,
      tooltipContent: expect.anything(),
    });

    expect(editConfig).toMatchObject({
      disabled: true,
      tooltipContent: expect.anything(),
    });

    // For Apple VPP, Edit software also gets the gitops tooltip if present.
    expect(editSoftware).toMatchObject({
      disabled: true,
      tooltipContent: expect.anything(),
    });
  });

  it("adds Deploy when software can be deployed", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canDeploySoftware: true,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
    });

    const deploy = result.find((opt) => opt.value === ACTION_DEPLOY);

    expect(deploy).toEqual({
      label: "Deploy",
      value: ACTION_DEPLOY,
    });
  });

  it("adds Versions option after Deploy when canManageVersions", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: true,
      canEditConfiguration: false,
      canDeploySoftware: true,
      canManageVersions: true,
      canConfigureAutoUpdate: false,
    });

    const values = result.map((o) => o.value);
    expect(values).toEqual([
      ACTION_EDIT_APPEARANCE,
      ACTION_EDIT_SOFTWARE,
      ACTION_DEPLOY,
      ACTION_VERSIONS,
    ]);

    const versions = result.find((opt) => opt.value === ACTION_VERSIONS);
    expect(versions).toEqual({
      label: "Versions",
      value: ACTION_VERSIONS,
    });
  });

  it("does not add Versions option when canManageVersions is false", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: false,
    });

    expect(result.find((o) => o.value === ACTION_VERSIONS)).toBeUndefined();
  });

  it("keeps Versions option enabled in GitOps mode (modal disables Save itself)", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: true,
      repoURL: "https://repo.git",
      canEditSoftware: false,
      canEditConfiguration: false,
      canDeploySoftware: false,
      canManageVersions: true,
      canConfigureAutoUpdate: false,
    });

    const versions = result.find((opt) => opt.value === ACTION_VERSIONS);
    expect(versions).toEqual({
      label: "Versions",
      value: ACTION_VERSIONS,
    });
  });

  it("adds Schedule auto updates option when canConfigureAutoUpdate", () => {
    const result = buildActionOptions({
      gitOpsModeEnabled: false,
      repoURL: undefined,
      canEditSoftware: false,
      canEditConfiguration: false,
      canDeploySoftware: false,
      canManageVersions: false,
      canConfigureAutoUpdate: true,
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

describe("SoftwareDetailsSummary headerPills slot", () => {
  it("renders headerPills content when provided", () => {
    render(
      <SoftwareDetailsSummary
        displayName="My software"
        headerPills={<span>marker-pill</span>}
      />
    );

    expect(screen.getByText("marker-pill")).toBeInTheDocument();
  });

  it("does not render the headerPills wrapper when not provided", () => {
    const { container } = render(
      <SoftwareDetailsSummary displayName="My software" />
    );

    expect(
      container.querySelector(".software-details-summary__header-pills")
    ).toBeNull();
  });
});

describe("SoftwareDetailsSummary IDs", () => {
  it("renders GitOps IDs when provided", () => {
    render(
      <SoftwareDetailsSummary
        displayName="My software"
        fleetMaintainedAppSlug="zoom/darwin"
        appStoreId="1016366447"
        googlePlayId="com.android.chrome"
      />
    );

    expect(screen.getByText("Fleet-maintained app slug")).toBeInTheDocument();
    expect(screen.getByText("zoom/darwin")).toBeInTheDocument();
    expect(screen.getByText("App Store ID")).toBeInTheDocument();
    expect(screen.getByText("1016366447")).toBeInTheDocument();
    expect(screen.getByText("Google Play ID")).toBeInTheDocument();
    expect(screen.getByText("com.android.chrome")).toBeInTheDocument();
  });

  it("does not render IDs that aren't provided", () => {
    render(<SoftwareDetailsSummary displayName="My software" />);

    expect(
      screen.queryByText("Fleet-maintained app slug")
    ).not.toBeInTheDocument();
    expect(screen.queryByText("App Store ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Google Play ID")).not.toBeInTheDocument();
  });
});

describe("SoftwareDetailsSummary VPP token", () => {
  const vppToken = {
    id: 2,
    org_name: "Acme K.K.",
    location: "Acme K.K.",
    country_code: "JP",
  };

  it("links the App Store ID to the token's storefront", () => {
    render(
      <SoftwareDetailsSummary
        displayName="PayPay"
        appStoreId="1435783608"
        vppToken={vppToken}
      />
    );

    expect(screen.getByRole("link", { name: /1435783608/ })).toHaveAttribute(
      "href",
      "https://apps.apple.com/jp/app/id1435783608"
    );
  });

  it("links the App Store ID without a country segment when there's no token", () => {
    render(
      <SoftwareDetailsSummary displayName="PayPay" appStoreId="1435783608" />
    );

    expect(screen.getByRole("link", { name: /1435783608/ })).toHaveAttribute(
      "href",
      "https://apps.apple.com/app/id1435783608"
    );
    expect(screen.queryByText("VPP token")).not.toBeInTheDocument();
  });

  it("renders the VPP token label and its tooltip copy", async () => {
    const { user } = renderWithSetup(
      <SoftwareDetailsSummary
        displayName="PayPay"
        appStoreId="1435783608"
        vppToken={vppToken}
      />
    );

    expect(screen.getByText("Acme K.K. • 🇯🇵 Japan")).toBeInTheDocument();

    await user.hover(screen.getByText("VPP token"));
    expect(
      await screen.findByText(
        "This is an organization unit in your Apple Business account, and licenses will be assigned from it."
      )
    ).toBeInTheDocument();
  });

  it("renders neither field for a title without an App Store ID or token", () => {
    render(<SoftwareDetailsSummary displayName="Chrome" />);

    expect(screen.queryByText("App Store ID")).not.toBeInTheDocument();
    expect(screen.queryByText("VPP token")).not.toBeInTheDocument();
  });
});
