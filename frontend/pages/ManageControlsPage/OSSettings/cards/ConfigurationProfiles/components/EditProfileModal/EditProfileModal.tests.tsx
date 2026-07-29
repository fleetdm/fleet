import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import { IMdmProfile } from "interfaces/mdm";
import labelsAPI from "services/entities/labels";
import mdmAPI from "services/entities/mdm";
import { notify } from "components/ToastNotification";

import EditProfileModal, {
  getAcceptedExtensions,
  getProfileFileExtension,
} from "./EditProfileModal";

const baseProfile: IMdmProfile = {
  profile_uuid: "abc-123",
  team_id: 0,
  name: "Test Profile",
  platform: "darwin",
  identifier: "com.example.test",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  checksum: null,
};

const mockLabels = [
  { id: 1, name: "Label A", description: "", label_type: "regular" as const },
  { id: 2, name: "Label B", description: "", label_type: "regular" as const },
];

const render = createCustomRenderer({ withBackendMock: true });

describe("EditProfileModal", () => {
  beforeEach(() => {
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: mockLabels });
    jest.spyOn(mdmAPI, "updateProfile").mockResolvedValue({
      profile_uuid: baseProfile.profile_uuid,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("renders the current profile's name and file extension without a target section on free tier", () => {
    render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier={false}
        onUpdate={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByText("Edit profile")).toBeInTheDocument();
    expect(screen.getByText("Test Profile")).toBeInTheDocument();
    expect(screen.getByText(".mobileconfig")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Update profile" })
    ).toBeEnabled();
    expect(screen.queryByText("Target")).not.toBeInTheDocument();
    expect(labelsAPI.summary).not.toHaveBeenCalled();
  });

  it("prefills a custom target from the profile's labels on premium tier", async () => {
    render(
      <EditProfileModal
        profile={{
          ...baseProfile,
          labels_include_all: [{ name: "Label A", id: 1 }],
        }}
        currentTeamId={0}
        isPremiumTier
        onUpdate={noop}
        onCancel={noop}
      />
    );

    await screen.findByText("Target");

    expect(screen.getByRole("radio", { name: "Custom" })).toBeChecked();
    // "All" include mode is preselected because the profile uses
    // labels_include_all.
    expect(document.getElementById("include-mode-all-radio")).toBeChecked();
    // the prefilled label renders both as a selected badge and as a checked
    // item in the label list
    expect(screen.getAllByText("Label A").length).toBeGreaterThan(0);
  });

  it("preselects All hosts when the profile has no labels", async () => {
    render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier
        onUpdate={noop}
        onCancel={noop}
      />
    );

    await screen.findByText("Target");

    expect(screen.getByRole("radio", { name: "All hosts" })).toBeChecked();
  });

  it("submits the full label selection without a file for a content-only edit", async () => {
    const onUpdate = jest.fn();
    const { user } = render(
      <EditProfileModal
        profile={{
          ...baseProfile,
          labels_include_all: [{ name: "Label A", id: 1 }],
        }}
        currentTeamId={0}
        isPremiumTier
        onUpdate={onUpdate}
        onCancel={noop}
      />
    );

    await screen.findByText("Target");
    await user.click(screen.getByRole("button", { name: "Update profile" }));

    await waitFor(() => {
      expect(mdmAPI.updateProfile).toHaveBeenCalledWith({
        profileUUID: "abc-123",
        profile: undefined,
        labelsIncludeAll: ["Label A"],
      });
    });
    expect(onUpdate).toHaveBeenCalledTimes(1);
  });

  it("submits no label fields when the target is switched to All hosts", async () => {
    const { user } = render(
      <EditProfileModal
        profile={{
          ...baseProfile,
          labels_exclude_any: [{ name: "Label B", id: 2 }],
        }}
        currentTeamId={0}
        isPremiumTier
        onUpdate={noop}
        onCancel={noop}
      />
    );

    await screen.findByText("Target");
    await user.click(screen.getByRole("radio", { name: "All hosts" }));
    await user.click(screen.getByRole("button", { name: "Update profile" }));

    await waitFor(() => {
      expect(mdmAPI.updateProfile).toHaveBeenCalledWith({
        profileUUID: "abc-123",
        profile: undefined,
      });
    });
  });

  it("disables the submit button when a custom target has no labels selected", async () => {
    const { user } = render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier
        onUpdate={noop}
        onCancel={noop}
      />
    );

    await screen.findByText("Target");
    await user.click(screen.getByRole("radio", { name: "Custom" }));

    expect(
      screen.getByRole("button", { name: "Update profile" })
    ).toBeDisabled();
    expect(mdmAPI.updateProfile).not.toHaveBeenCalled();
  });

  it("rejects a file extension valid for other profile types but not this one, leaving the displayed file unchanged", async () => {
    const errorSpy = jest.spyOn(notify, "error");
    const { container } = render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier={false}
        onUpdate={noop}
        onCancel={noop}
      />
    );

    const fileInput = container.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement;
    // .json is valid for Android profiles and DDM declarations but not for
    // baseProfile, an Apple config profile -- this proves the per-platform
    // check is enforced, not just a blanket file-type sniff. fireEvent
    // bypasses the input's `accept` filtering (which user-event enforces),
    // mirroring a user picking "All Files" in the OS dialog.
    const badFile = new File(["{}"], "bad.json", {
      type: "application/json",
    });
    fireEvent.change(fileInput, { target: { files: [badFile] } });

    await waitFor(() => {
      expect(errorSpy).toHaveBeenCalledWith(
        "Invalid file type",
        expect.anything()
      );
    });
    expect(screen.getByText("Test Profile")).toBeInTheDocument();
    expect(screen.getByText(".mobileconfig")).toBeInTheDocument();
  });

  it("accepts a valid replacement file and updates the displayed file details", async () => {
    const { user, container } = render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier={false}
        onUpdate={noop}
        onCancel={noop}
      />
    );

    const fileInput = container.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement;
    const newFile = new File(["<plist></plist>"], "new-profile.mobileconfig", {
      type: "application/x-apple-aspen-config",
    });
    await user.upload(fileInput, newFile);

    await waitFor(() => {
      expect(screen.getByText("new-profile")).toBeInTheDocument();
    });
    expect(screen.getByText(".mobileconfig")).toBeInTheDocument();
  });

  it("calls onCancel when cancel is clicked", async () => {
    const onCancel = jest.fn();
    const { user } = render(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier={false}
        onUpdate={noop}
        onCancel={onCancel}
      />
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("disables the submit button in GitOps mode", () => {
    const renderWithGitOps = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "https://github.com/example/fleet-gitops",
            },
          },
        },
      },
    });

    renderWithGitOps(
      <EditProfileModal
        profile={baseProfile}
        currentTeamId={0}
        isPremiumTier={false}
        onUpdate={noop}
        onCancel={noop}
      />
    );

    expect(
      screen.getByRole("button", { name: "Update profile" })
    ).toBeDisabled();
  });
});

describe("getAcceptedExtensions", () => {
  it("accepts .mobileconfig and .xml for Apple configuration profiles", () => {
    expect(getAcceptedExtensions(baseProfile)).toEqual([
      ".mobileconfig",
      ".xml",
    ]);
  });

  it("accepts only .json for Apple DDM declarations", () => {
    expect(
      getAcceptedExtensions({ ...baseProfile, profile_uuid: "d-abc-123" })
    ).toEqual([".json"]);
  });

  it("accepts only .xml for Windows profiles", () => {
    expect(
      getAcceptedExtensions({
        ...baseProfile,
        profile_uuid: "w-abc-123",
        platform: "windows",
      })
    ).toEqual([".xml"]);
  });

  it("accepts only .json for Android profiles", () => {
    expect(
      getAcceptedExtensions({
        ...baseProfile,
        profile_uuid: "g-abc-123",
        platform: "android",
      })
    ).toEqual([".json"]);
  });

  it("accepts nothing for an unknown platform", () => {
    expect(
      getAcceptedExtensions({ ...baseProfile, platform: "linux" })
    ).toEqual([]);
  });
});

describe("getProfileFileExtension", () => {
  it("returns .mobileconfig for Apple configuration profiles", () => {
    expect(getProfileFileExtension(baseProfile)).toEqual(".mobileconfig");
  });

  it("returns .json for Apple DDM declarations", () => {
    expect(
      getProfileFileExtension({ ...baseProfile, profile_uuid: "d-abc-123" })
    ).toEqual(".json");
  });

  it("returns .xml for Windows profiles", () => {
    expect(
      getProfileFileExtension({
        ...baseProfile,
        profile_uuid: "w-abc-123",
        platform: "windows",
      })
    ).toEqual(".xml");
  });

  it("returns .json for Android profiles", () => {
    expect(
      getProfileFileExtension({
        ...baseProfile,
        profile_uuid: "g-abc-123",
        platform: "android",
      })
    ).toEqual(".json");
  });

  it("returns no extension for an unknown platform", () => {
    expect(
      getProfileFileExtension({ ...baseProfile, platform: "linux" })
    ).toEqual("");
  });
});
