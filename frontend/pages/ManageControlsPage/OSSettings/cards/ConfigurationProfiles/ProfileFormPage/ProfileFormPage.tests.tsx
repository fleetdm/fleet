import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
import { IMdmProfile } from "interfaces/mdm";
import configProfileAPI from "services/entities/config_profiles";
import labelsAPI from "services/entities/labels";
import mdmAPI from "services/entities/mdm";
import {
  createCustomRenderer,
  createMockLocation,
  createMockRouter,
} from "test/test-utils";

import ProfileFormPage from "./ProfileFormPage";

// Drive the resolved fleet directly, as ManageControlsPage.tests does: the
// real hook needs a routed app to settle on a fleet.
const mockUseTeamIdParam = jest.fn();
jest.mock("hooks/useTeamIdParam", () => ({
  __esModule: true,
  default: () => mockUseTeamIdParam(),
}));

// Ace doesn't take input under jsdom, so stand in a plain textarea with the
// same accessible name, label (replaced by the error, as the real one does),
// help text and handlers.
jest.mock("components/Editor", () => ({
  __esModule: true,
  default: ({
    label,
    ariaLabel,
    error,
    helpText,
    value,
    onChange,
    onFocus,
    onBlur,
    readOnly,
  }: {
    label?: string;
    ariaLabel?: string;
    error?: string | null;
    helpText?: React.ReactNode;
    value?: string;
    onChange?: (value: string) => void;
    onFocus?: () => void;
    onBlur?: () => void;
    readOnly?: boolean;
  }) => (
    <div>
      {(error || label) && <div>{error || label}</div>}
      <textarea
        aria-label={ariaLabel ?? label}
        value={value}
        readOnly={readOnly}
        onChange={(e) => onChange?.(e.target.value)}
        onFocus={onFocus}
        onBlur={onBlur}
      />
      {helpText && <div>{helpText}</div>}
    </div>
  ),
}));

const MOBILECONFIG = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDisplayName</key>
  <string>Payload Name</string>
  <key>PayloadIdentifier</key>
  <string>com.example.test</string>
</dict>
</plist>`;

const WINDOWS_XML = `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Test</LocURI></Target></Item></Replace>`;

const existingProfile: IMdmProfile = {
  profile_uuid: "w-123",
  team_id: 0,
  name: "Existing Windows",
  description: "Existing description",
  platform: "windows",
  identifier: null,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  checksum: null,
  self_service: false,
  hidden: false,
};

const mdmConfig = {
  ...createMockConfig().mdm,
  enabled_and_configured: true,
  windows_enabled_and_configured: true,
  android_enabled_and_configured: true,
};

const makeRenderer = (
  appOverrides: Record<string, unknown> = {},
  gitOpsModeEnabled = false,
  mdm = mdmConfig
) =>
  createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: false,
        isGlobalAdmin: true,
        currentUser: createMockUser(),
        currentTeam: createMockTeamSummary({ id: 0, name: "No fleet" }),
        availableTeams: [createMockTeamSummary({ id: 0, name: "No fleet" })],
        config: createMockConfig({
          mdm,
          gitops: {
            ...createMockConfig().gitops,
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: gitOpsModeEnabled ? "https://example.com/repo" : "",
          },
        }),
        ...appOverrides,
      },
    },
  });

const render = makeRenderer();

const renderPage = (
  profileUUID?: string,
  customRender = render,
  query: Record<string, string> = {}
) => {
  const router = createMockRouter();
  const location = createMockLocation({
    pathname: profileUUID
      ? `/controls/os-settings/configuration-profiles/${profileUUID}`
      : "/controls/os-settings/configuration-profiles/new",
    search: Object.keys(query).length
      ? `?${new URLSearchParams(query).toString()}`
      : "",
    query,
  });
  // the remaining RouteComponentProps are unused by the page
  const props = ({
    router,
    location,
    routeParams: { profile_uuid: profileUUID },
  } as unknown) as React.ComponentProps<typeof ProfileFormPage>;
  const results = customRender(<ProfileFormPage {...props} />);
  return { ...results, router };
};

/** The multipart file is built from the editor contents. jsdom's File has
 * no text(), so go through FileReader. */
const readFile = (file?: File) =>
  new Promise<string>((resolve) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result));
    reader.readAsText(file ?? new Blob());
  });

/** jsdom's File has no text(), which the page reads uploads with. */
const fileWithText = (contents: string, name: string, readFails = false) => {
  const file = new File([contents], name);
  Object.defineProperty(file, "text", {
    value: () =>
      readFails
        ? Promise.reject(new Error("read failed"))
        : Promise.resolve(contents),
  });
  return file;
};

const setContents = (value: string) =>
  fireEvent.change(screen.getByLabelText("Profile contents"), {
    target: { value },
  });

describe("ProfileFormPage", () => {
  beforeEach(() => {
    mockUseTeamIdParam.mockReturnValue({
      currentTeamId: 0,
      currentTeamName: "No fleet",
      teamIdForApi: 0,
      userTeams: [createMockTeamSummary({ id: 0, name: "No fleet" })],
      handleTeamChange: jest.fn(),
    });
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: [] });
    jest
      .spyOn(mdmAPI, "uploadProfile")
      .mockResolvedValue({ profile_uuid: "a-new" });
    jest
      .spyOn(mdmAPI, "updateProfile")
      .mockResolvedValue({ profile_uuid: existingProfile.profile_uuid });
    jest
      .spyOn(configProfileAPI, "getConfigProfile")
      .mockResolvedValue(existingProfile);
    jest.spyOn(mdmAPI, "downloadProfile").mockResolvedValue(WINDOWS_XML);
    jest.spyOn(mdmAPI, "getProfiles").mockResolvedValue({
      profiles: [],
      meta: { has_next_results: false, has_previous_results: false },
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("keeps Add profile disabled until there's a profile, reports unrecognized contents on submit, then uploads", async () => {
    const { user, router } = renderPage();

    expect(
      screen.getByRole("heading", { name: "Add profile" })
    ).toBeInTheDocument();
    const submit = screen.getByRole("button", { name: "Add profile" });
    // per the design, the empty add page is the one disabled state
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText("Name"), "Custom Apple");
    await user.type(screen.getByLabelText("Description"), "Blocks the camera");
    expect(submit).toBeDisabled();

    // anything pasted enables it; unrecognized text is reported on submit
    setContents("not a profile");
    expect(submit).toBeEnabled();
    await user.click(submit);
    expect(
      screen.getByText(/^Paste a .mobileconfig, declaration/)
    ).toBeInTheDocument();
    expect(mdmAPI.uploadProfile).not.toHaveBeenCalled();

    setContents(MOBILECONFIG);
    await user.click(submit);
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    expect(screen.getByText("Mobileconfig")).toBeInTheDocument();
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.name).toBe("Custom Apple");
    expect(args.description).toBe("Blocks the camera");
    expect(args.file.name).toBe("New profile.mobileconfig");
    expect(await readFile(args.file)).toContain("Payload Name");
    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles"
    );
  });

  it("omits name and description when left blank so the server derives them", async () => {
    const { user } = renderPage();

    setContents(WINDOWS_XML);
    expect(screen.getByText("Windows")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.name).toBeUndefined();
    expect(args.description).toBeUndefined();
    expect(args.file.name).toBe("New profile.xml");
  });

  it("names an unnamed pasted profile after the next free default", async () => {
    jest.mocked(mdmAPI.getProfiles).mockResolvedValue({
      profiles: [
        { ...existingProfile, name: "New profile" },
        { ...existingProfile, profile_uuid: "w-456", name: "new profile 3" },
      ],
      meta: { has_next_results: false, has_previous_results: false },
    });
    const { user } = renderPage();

    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    expect(mdmAPI.getProfiles).toHaveBeenCalledTimes(1);
    expect(mdmAPI.getProfiles).toHaveBeenCalledWith({
      fleet_id: 0,
      page: 0,
      per_page: 1000,
    });
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.name).toBeUndefined();
    expect(args.file.name).toBe("New profile 2.xml");
  });

  it("seeds the edit form and sends only the changed metadata, without a file", async () => {
    const { user, router } = renderPage(existingProfile.profile_uuid);

    expect(await screen.findByText("Edit profile")).toBeInTheDocument();
    const nameInput = await screen.findByLabelText("Name");
    await waitFor(() => expect(nameInput).toHaveValue("Existing Windows"));
    expect(screen.getByLabelText("Description")).toHaveValue(
      "Existing description"
    );
    expect(screen.getByText("Windows")).toBeInTheDocument();
    // fetched as text so JSON profiles aren't re-serialized
    expect(mdmAPI.downloadProfile).toHaveBeenCalledTimes(1);
    expect(mdmAPI.downloadProfile).toHaveBeenCalledWith(
      existingProfile.profile_uuid,
      "text"
    );

    const submit = screen.getByRole("button", { name: "Update profile" });
    // a no-op re-save is allowed
    expect(submit).toBeEnabled();

    await user.clear(nameInput);
    await user.type(nameInput, "Renamed Windows");
    await user.clear(screen.getByLabelText("Description"));
    expect(submit).toBeEnabled();

    await user.click(submit);
    await waitFor(() => expect(mdmAPI.updateProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.updateProfile).mock.calls[0][0];
    expect(args.profileUUID).toBe(existingProfile.profile_uuid);
    expect(args.name).toBe("Renamed Windows");
    // cleared, so an empty string is sent to clear it server-side
    expect(args.description).toBe("");
    expect(args.profile).toBeUndefined();
    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles"
    );
  });

  it("goes back without a request when nothing changed", async () => {
    const successSpy = jest.spyOn(notify, "success");
    const { user, router } = renderPage(existingProfile.profile_uuid);
    expect(await screen.findByText("Edit profile")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByLabelText("Name")).toHaveValue("Existing Windows")
    );

    await user.click(screen.getByRole("button", { name: "Update profile" }));

    await waitFor(() => expect(router.push).toHaveBeenCalledTimes(1));
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles"
    );
    expect(mdmAPI.updateProfile).not.toHaveBeenCalled();
    expect(successSpy).not.toHaveBeenCalled();
  });

  it("sends a replacement file when the contents change and keeps the name", async () => {
    const { user } = renderPage(existingProfile.profile_uuid);
    expect(await screen.findByText("Edit profile")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByLabelText("Name")).toHaveValue("Existing Windows")
    );

    setContents("<Add/>");

    await user.click(screen.getByRole("button", { name: "Update profile" }));
    await waitFor(() => expect(mdmAPI.updateProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.updateProfile).mock.calls[0][0];
    expect(args.name).toBeUndefined();
    expect(args.description).toBeUndefined();
    expect(args.profile).toBeInstanceOf(File);
    expect(args.profile?.name).toBe("New profile.xml");
    expect(await readFile(args.profile)).toContain("<Add/>");
  });

  it("disables every field and the submit button in GitOps mode", async () => {
    renderPage(existingProfile.profile_uuid, makeRenderer({}, true));
    expect(await screen.findByText("Edit profile")).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByLabelText("Name")).toHaveValue("Existing Windows")
    );

    expect(screen.getByLabelText("Name")).toBeDisabled();
    expect(screen.getByLabelText("Description")).toBeDisabled();
    expect(screen.getByLabelText("Upload a profile")).toBeDisabled();
    expect(screen.getByLabelText("Profile contents")).toHaveAttribute(
      "readonly"
    );
    expect(
      screen.getByRole("button", { name: "Update profile" })
    ).toBeDisabled();
  });

  it("refuses users who aren't admins or maintainers", () => {
    renderPage(
      undefined,
      makeRenderer({ isGlobalAdmin: false, isGlobalTechnician: true })
    );
    expect(
      screen.getByText("You don't have permission to edit profiles.")
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
  });

  it("reports a blank name on an existing profile instead of saving it", async () => {
    const { user } = renderPage(existingProfile.profile_uuid);
    const nameInput = await screen.findByLabelText("Name");
    await waitFor(() => expect(nameInput).toHaveValue("Existing Windows"));

    await user.clear(nameInput);
    await user.click(screen.getByRole("button", { name: "Update profile" }));
    expect(screen.getByText("Enter a name")).toBeInTheDocument();
    expect(mdmAPI.updateProfile).not.toHaveBeenCalled();

    // focusing the field clears its error
    await user.type(nameInput, "Named again");
    expect(screen.queryByText("Enter a name")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Update profile" }));
    await waitFor(() => expect(mdmAPI.updateProfile).toHaveBeenCalledTimes(1));
    expect(jest.mocked(mdmAPI.updateProfile).mock.calls[0][0].name).toBe(
      "Named again"
    );
  });

  it("reports cleared contents on an existing profile instead of saving it", async () => {
    const { user } = renderPage(existingProfile.profile_uuid);
    await waitFor(() =>
      expect(screen.getByLabelText("Profile contents")).toHaveValue(WINDOWS_XML)
    );

    setContents("");
    await user.click(screen.getByRole("button", { name: "Update profile" }));

    expect(screen.getByText("Upload or paste a profile")).toBeInTheDocument();
    expect(mdmAPI.updateProfile).not.toHaveBeenCalled();
  });

  it("shows the Target selector on Premium and reports a custom target without labels", async () => {
    const { user } = renderPage(
      undefined,
      makeRenderer({
        isPremiumTier: true,
        currentTeam: createMockTeamSummary({ id: 1, name: "Workstations" }),
        availableTeams: [
          createMockTeamSummary({ id: 1, name: "Workstations" }),
        ],
      })
    );
    expect(await screen.findByText("Target")).toBeInTheDocument();
    setContents(WINDOWS_XML);
    const submit = screen.getByRole("button", { name: "Add profile" });
    await waitFor(() => expect(submit).toBeEnabled());

    await user.click(screen.getByLabelText("Custom"));
    expect(screen.getByText(/Applies to/)).toHaveTextContent(
      "hosts in custom targets"
    );
    expect(submit).toBeEnabled();
    await user.click(submit);
    expect(screen.getByText("Select at least one label")).toBeInTheDocument();
    expect(mdmAPI.uploadProfile).not.toHaveBeenCalled();

    // going back to All hosts makes the error irrelevant
    await user.click(screen.getByLabelText("All hosts"));
    expect(
      screen.queryByText("Select at least one label")
    ).not.toBeInTheDocument();
  });

  it("disables the add form when no MDM is turned on", () => {
    renderPage(
      undefined,
      makeRenderer({}, false, {
        ...mdmConfig,
        enabled_and_configured: false,
        windows_enabled_and_configured: false,
        android_enabled_and_configured: false,
      })
    );
    expect(screen.getByLabelText("Name")).toBeDisabled();
    expect(screen.getByLabelText("Upload a profile")).toBeDisabled();
    expect(screen.getByLabelText("Profile contents")).toHaveAttribute(
      "readonly"
    );
    expect(screen.getByRole("button", { name: "Add profile" })).toBeDisabled();
  });

  it("blocks adding a pasted profile whose platform's MDM is off", async () => {
    renderPage(
      undefined,
      makeRenderer({}, false, {
        ...mdmConfig,
        enabled_and_configured: true,
        windows_enabled_and_configured: false,
        android_enabled_and_configured: false,
      })
    );
    const submit = screen.getByRole("button", { name: "Add profile" });

    setContents(WINDOWS_XML);
    expect(submit).toBeDisabled();
    // still editable, so the admin can paste something else
    expect(screen.getByLabelText("Profile contents")).not.toHaveAttribute(
      "readonly"
    );

    setContents(MOBILECONFIG);
    // the button remounts once the MDM tooltip no longer wraps it
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Add profile" })).toBeEnabled()
    );
  });

  it("gates JSON with uppercase keys on Apple MDM, as the server types it", () => {
    renderPage(
      undefined,
      makeRenderer({}, false, {
        ...mdmConfig,
        enabled_and_configured: true,
        windows_enabled_and_configured: false,
        android_enabled_and_configured: false,
      })
    );

    // missing Type, which the server reports for a declaration
    setContents('{"Identifier": "x", "Payload": {}}');

    expect(screen.getByText("Declaration (DDM)")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add profile" })).toBeEnabled();
  });

  it("sends SyncML behind an XML declaration as Windows", async () => {
    const { user } = renderPage(
      undefined,
      makeRenderer({}, false, {
        ...mdmConfig,
        enabled_and_configured: false,
        windows_enabled_and_configured: true,
        android_enabled_and_configured: false,
      })
    );

    setContents(`<?xml version="1.0" encoding="UTF-8"?>\n${WINDOWS_XML}`);
    expect(screen.getByText("Windows")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.file.name).toBe("New profile.xml");
  });

  it("follows the profile's fleet when the URL names a different one", async () => {
    mockUseTeamIdParam.mockReturnValue({
      currentTeamId: 1,
      currentTeamName: "Workstations",
      teamIdForApi: 1,
      userTeams: [
        createMockTeamSummary({ id: 1, name: "Workstations" }),
        createMockTeamSummary({ id: 3, name: "Servers" }),
      ],
      handleTeamChange: jest.fn(),
    });
    jest
      .spyOn(configProfileAPI, "getConfigProfile")
      .mockResolvedValue({ ...existingProfile, team_id: 3 });

    const { router } = renderPage(
      existingProfile.profile_uuid,
      makeRenderer({
        isPremiumTier: true,
        currentTeam: createMockTeamSummary({ id: 1, name: "Workstations" }),
        availableTeams: [
          createMockTeamSummary({ id: 1, name: "Workstations" }),
          createMockTeamSummary({ id: 3, name: "Servers" }),
        ],
      }),
      { fleet_id: "1" }
    );

    await waitFor(() =>
      expect(router.replace).toHaveBeenCalledWith(
        `/controls/os-settings/configuration-profiles/${existingProfile.profile_uuid}?fleet_id=3`
      )
    );
    expect(router.replace).toHaveBeenCalledTimes(1);
    // the wrong fleet's form is never shown
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
  });

  it("refuses a profile in a fleet the user can't manage instead of following it", async () => {
    // an admin of Workstations who is only a technician of Servers
    mockUseTeamIdParam.mockReturnValue({
      currentTeamId: 1,
      currentTeamName: "Workstations",
      teamIdForApi: 1,
      userTeams: [createMockTeamSummary({ id: 1, name: "Workstations" })],
      handleTeamChange: jest.fn(),
    });
    jest
      .spyOn(configProfileAPI, "getConfigProfile")
      .mockResolvedValue({ ...existingProfile, team_id: 3 });

    const { router } = renderPage(
      existingProfile.profile_uuid,
      makeRenderer({
        isPremiumTier: true,
        isGlobalAdmin: false,
        isAnyTeamMaintainerOrTeamAdmin: true,
        currentTeam: createMockTeamSummary({ id: 1, name: "Workstations" }),
      }),
      { fleet_id: "3" }
    );

    expect(
      await screen.findByText("You don't have permission to edit this profile.")
    ).toBeInTheDocument();
    expect(router.replace).not.toHaveBeenCalled();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
  });

  it("uploads a picked file and names the profile after it", async () => {
    const { user } = renderPage();

    await user.upload(
      screen.getByLabelText("Upload a profile"),
      fileWithText(WINDOWS_XML, "Firewall.xml")
    );
    await waitFor(() =>
      expect(screen.getByLabelText("Profile contents")).toHaveValue(WINDOWS_XML)
    );
    expect(screen.getByText("Windows")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.name).toBeUndefined();
    expect(args.file.name).toBe("Firewall.xml");
    // a picked file already carries a name, so no default is looked up
    expect(mdmAPI.getProfiles).not.toHaveBeenCalled();
  });

  it("types a picked file the contents can't type by its extension", async () => {
    const { user } = renderPage();
    setContents("not a profile");
    await user.click(screen.getByRole("button", { name: "Add profile" }));
    expect(
      await screen.findByText(/^Paste a .mobileconfig, declaration/)
    ).toBeInTheDocument();

    await user.upload(
      screen.getByLabelText("Upload a profile"),
      fileWithText("$FLEET_SECRET_PROFILE", "Secret.xml")
    );
    await waitFor(() =>
      expect(screen.getByLabelText("Profile contents")).toHaveValue(
        "$FLEET_SECRET_PROFILE"
      )
    );
    expect(
      screen.queryByText(/^Paste a .mobileconfig, declaration/)
    ).not.toBeInTheDocument();
    expect(screen.getByText("Windows")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.file.name).toBe("Secret.xml");
  });

  it("stops typing contents by the picked file once they're edited", async () => {
    const { user } = renderPage();

    await user.upload(
      screen.getByLabelText("Upload a profile"),
      fileWithText("$FLEET_SECRET_PROFILE", "Secret.xml")
    );
    await waitFor(() =>
      expect(screen.getByLabelText("Profile contents")).toHaveValue(
        "$FLEET_SECRET_PROFILE"
      )
    );
    setContents("$FLEET_SECRET_OTHER");

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    expect(
      await screen.findByText(/^Paste a .mobileconfig, declaration/)
    ).toBeInTheDocument();
    expect(mdmAPI.uploadProfile).not.toHaveBeenCalled();
  });

  it("rejects a replacement file of another type", async () => {
    const errorSpy = jest.spyOn(notify, "error");
    renderPage(existingProfile.profile_uuid);
    await waitFor(() =>
      expect(screen.getByLabelText("Name")).toHaveValue("Existing Windows")
    );

    // bypasses the picker's accept filter, as a drag and drop would
    fireEvent.change(screen.getByLabelText("Upload a profile"), {
      target: { files: [fileWithText("{}", "profile.json")] },
    });

    await waitFor(() => expect(errorSpy).toHaveBeenCalledTimes(1));
    expect(errorSpy).toHaveBeenCalledWith(
      "Invalid file type",
      expect.anything()
    );
    expect(screen.getByLabelText("Profile contents")).toHaveValue(WINDOWS_XML);
  });

  it("says so when a picked file can't be read", async () => {
    const errorSpy = jest.spyOn(notify, "error");
    const { user } = renderPage();

    await user.upload(
      screen.getByLabelText("Upload a profile"),
      fileWithText(WINDOWS_XML, "Firewall.xml", true)
    );

    await waitFor(() => expect(errorSpy).toHaveBeenCalledTimes(1));
    expect(errorSpy).toHaveBeenCalledWith(
      "Couldn't read the file. Please try again.",
      expect.anything()
    );
    expect(screen.getByLabelText("Profile contents")).toHaveValue("");
  });

  it("shows an error instead of the form when the profile can't be loaded", async () => {
    // a 4xx, as the API client rejects with, so the query doesn't retry
    jest
      .spyOn(configProfileAPI, "getConfigProfile")
      .mockRejectedValue({ status: 404 });
    renderPage(existingProfile.profile_uuid);

    expect(
      await screen.findByText("Couldn't load the profile.")
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
  });

  it("shows a name clash on the Name field and toasts it", async () => {
    const reason =
      "A configuration profile with this name already exists. Enter a different name.";
    jest
      .mocked(mdmAPI.uploadProfile)
      .mockRejectedValue({ data: { errors: [{ name: "profile", reason }] } });
    const errorSpy = jest.spyOn(notify, "error");
    const { user, router } = renderPage();

    await user.type(screen.getByLabelText("Name"), "Taken");
    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));

    await waitFor(() => expect(errorSpy).toHaveBeenCalledTimes(1));
    expect(errorSpy).toHaveBeenCalledWith(`Couldn't add. ${reason}`);
    // the field's label shows the error
    expect(screen.getByText(`Couldn't add. ${reason}`)).toBeInTheDocument();
    expect(router.push).not.toHaveBeenCalled();
  });

  it("toasts any other server error and stays on the form", async () => {
    jest.mocked(mdmAPI.uploadProfile).mockRejectedValue({
      data: {
        errors: [
          { name: "base", reason: "The profile should include valid JSON." },
        ],
      },
    });
    const errorSpy = jest.spyOn(notify, "error");
    const { user, router } = renderPage();

    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));

    await waitFor(() => expect(errorSpy).toHaveBeenCalledTimes(1));
    expect(errorSpy).toHaveBeenCalledWith(
      "Couldn't add. The profile should include valid JSON.",
      expect.anything()
    );
    expect(router.push).not.toHaveBeenCalled();
  });

  it("disables the fields and the button while saving", async () => {
    jest
      .mocked(mdmAPI.uploadProfile)
      .mockReturnValue(new Promise(() => undefined));
    const { user } = renderPage();

    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));

    await waitFor(() => expect(screen.getByLabelText("Name")).toBeDisabled());
    expect(screen.getByLabelText("Profile contents")).toHaveAttribute(
      "readonly"
    );
    expect(screen.getByRole("button", { name: /Add profile/ })).toBeDisabled();
  });

  it("keeps the leave-page prompt while saving", async () => {
    jest
      .mocked(mdmAPI.uploadProfile)
      .mockReturnValue(new Promise(() => undefined));
    const { user } = renderPage();

    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(screen.getByLabelText("Name")).toBeDisabled());

    const leave = new Event("beforeunload", { cancelable: true });
    window.dispatchEvent(leave);
    expect(leave.defaultPrevented).toBe(true);
  });

  it("falls back to the plain default when the name lookup fails", async () => {
    jest.mocked(mdmAPI.getProfiles).mockRejectedValue(new Error("offline"));
    const { user } = renderPage();

    setContents(WINDOWS_XML);
    await user.click(screen.getByRole("button", { name: "Add profile" }));

    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.file.name).toBe("New profile.xml");
  });

  it("goes back to the profiles list on Cancel", async () => {
    const { user, router } = renderPage();

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(router.push).toHaveBeenCalledTimes(1);
    expect(router.push).toHaveBeenCalledWith(
      "/controls/os-settings/configuration-profiles"
    );
  });

  describe("Deploy", () => {
    const premiumRender = makeRenderer({ isPremiumTier: true });
    const appleProfile: IMdmProfile = {
      ...existingProfile,
      profile_uuid: "a-123",
      name: "Existing Apple",
      platform: "darwin",
      identifier: "com.example.test",
      self_service: true,
    };

    const hiddenCheckbox = () =>
      screen.queryByRole("checkbox", { name: "Hide from end user" });
    const chooseDeploy = async (
      user: ReturnType<typeof renderPage>["user"],
      option: string
    ) => {
      await user.click(screen.getByRole("combobox"));
      // the options have no option role
      await user.click(await screen.findByText(option));
    };
    // DropdownWrapper remounts its input on every render, which closes an
    // open menu, so let the labels query settle before opening it.
    const waitForEditForm = async () => {
      await waitFor(() =>
        expect(screen.getByLabelText("Name")).toHaveValue("Existing Apple")
      );
      await waitFor(() => expect(labelsAPI.summary).toHaveBeenCalled());
      await act(async () => {
        await Promise.resolve();
      });
    };

    it("offers self-service for a .mobileconfig and never sends it hidden", async () => {
      const { user } = renderPage(undefined, premiumRender);
      setContents(MOBILECONFIG);

      expect(screen.getByText("Force install")).toBeInTheDocument();
      await user.click(hiddenCheckbox() as HTMLElement);

      await chooseDeploy(user, "End user initiated (manual)");
      expect(hiddenCheckbox()).not.toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Add profile" }));
      await waitFor(() =>
        expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1)
      );
      const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
      expect(args.selfService).toBe(true);
      expect(args.hidden).toBe(false);
    });

    it("resets self-service when the uploaded file is replaced with another type", async () => {
      const { user } = renderPage(undefined, premiumRender);
      const upload = async (contents: string, name: string) => {
        await user.upload(
          screen.getByLabelText("Upload a profile"),
          fileWithText(contents, name)
        );
        await waitFor(() =>
          expect(screen.getByLabelText("Profile contents")).toHaveValue(
            contents
          )
        );
      };

      await upload(MOBILECONFIG, "Opt-in.mobileconfig");
      await chooseDeploy(user, "End user initiated (manual)");
      await upload(WINDOWS_XML, "Firewall.xml");
      await upload(MOBILECONFIG, "Opt-in.mobileconfig");
      expect(screen.getByText("Force install")).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Add profile" }));
      await waitFor(() =>
        expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1)
      );
      expect(
        jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0].selfService
      ).toBe(false);
    });

    it("is force-only for other types, which can still be hidden", async () => {
      const { user } = renderPage(undefined, premiumRender);
      setContents(WINDOWS_XML);

      expect(screen.getByText("Force install")).toBeInTheDocument();
      expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
      await user.click(hiddenCheckbox() as HTMLElement);

      await user.click(screen.getByRole("button", { name: "Add profile" }));
      await waitFor(() =>
        expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1)
      );
      const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
      expect(args.selfService).toBe(false);
      expect(args.hidden).toBe(true);
    });

    it("sends neither flag on Free", async () => {
      const { user } = renderPage();
      setContents(MOBILECONFIG);
      expect(screen.queryByText("Deploy")).not.toBeInTheDocument();
      expect(hiddenCheckbox()).not.toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: "Add profile" }));
      await waitFor(() =>
        expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1)
      );
      const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
      expect(args.selfService).toBeUndefined();
      expect(args.hidden).toBeUndefined();
    });

    it("seeds an edit from the profile and saves a deploy-only change without a file", async () => {
      jest
        .spyOn(configProfileAPI, "getConfigProfile")
        .mockResolvedValue(appleProfile);
      jest.spyOn(mdmAPI, "downloadProfile").mockResolvedValue(MOBILECONFIG);
      const { user } = renderPage(appleProfile.profile_uuid, premiumRender);
      await waitForEditForm();
      expect(
        screen.getByText("End user initiated (manual)")
      ).toBeInTheDocument();
      expect(hiddenCheckbox()).not.toBeInTheDocument();

      await chooseDeploy(user, "Force install");
      await user.click(hiddenCheckbox() as HTMLElement);

      await user.click(screen.getByRole("button", { name: "Update profile" }));
      await waitFor(() =>
        expect(mdmAPI.updateProfile).toHaveBeenCalledTimes(1)
      );
      const args = jest.mocked(mdmAPI.updateProfile).mock.calls[0][0];
      expect(args.profile).toBeUndefined();
      expect(args.selfService).toBe(false);
      expect(args.hidden).toBe(true);
    });

    it("disables both controls in GitOps mode", async () => {
      jest
        .spyOn(configProfileAPI, "getConfigProfile")
        .mockResolvedValue({ ...appleProfile, self_service: false });
      jest.spyOn(mdmAPI, "downloadProfile").mockResolvedValue(MOBILECONFIG);
      renderPage(
        appleProfile.profile_uuid,
        makeRenderer({ isPremiumTier: true }, true)
      );
      await waitForEditForm();
      expect(screen.getByText("Force install")).toBeInTheDocument();
      expect(screen.getByRole("combobox")).toBeDisabled();
      expect(hiddenCheckbox()).toHaveAttribute("aria-disabled", "true");
    });
  });
});
