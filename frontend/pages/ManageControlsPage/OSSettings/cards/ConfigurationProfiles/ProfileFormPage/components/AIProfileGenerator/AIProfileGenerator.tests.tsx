import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import autofillAPI, {
  IAutofillConfigProfileContentResponse,
  IAutofillConfigProfileNameDescriptionResponse,
} from "services/entities/autofill";
import labelsAPI from "services/entities/labels";
import mdmAPI from "services/entities/mdm";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import AIProfileGenerator from "./AIProfileGenerator";
import { GENERATION_TIMEOUT_MS, READING_STEP_MS } from "./useProfileGeneration";

// Ace doesn't render under jsdom; a textarea with the same name and help
// text stands in, as in ProfileFormPage.tests.
jest.mock("components/Editor", () => ({
  __esModule: true,
  default: ({
    label,
    ariaLabel,
    helpText,
    value,
    onChange,
  }: {
    label?: string;
    ariaLabel?: string;
    helpText?: React.ReactNode;
    value?: string;
    onChange?: (value: string) => void;
  }) => (
    <div>
      {label && <div>{label}</div>}
      <textarea
        aria-label={ariaLabel ?? label}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
      {helpText && <div>{helpText}</div>}
    </div>
  ),
}));

const MOBILECONFIG = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict><key>PayloadDisplayName</key><string>AirDrop Disabled</string></dict></plist>`;

const MOBILECONFIG_RESPONSE: IAutofillConfigProfileContentResponse = {
  type: "mobileconfig",
  name: "AirDrop Disabled",
  content: MOBILECONFIG,
  delivery_notes: "",
};

const DECLARATION_RESPONSE: IAutofillConfigProfileContentResponse = {
  type: "declaration",
  name: null,
  content: `{"Type":"com.apple.configuration.passcode.settings"}`,
  delivery_notes: "",
};

const NAME_DESCRIPTION: IAutofillConfigProfileNameDescriptionResponse = {
  name: "Disable AirDrop on Macs",
  description: "Sets `allowAirDrop` to `false`.",
};

interface IDeferred<T> {
  promise: Promise<T>;
  resolve: (value: T) => void;
  reject: (reason: unknown) => void;
}

function deferred<T>(): IDeferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

const apiError = (name: string, reason: string) => ({
  status: 422,
  data: { message: "Couldn't generate", errors: [{ name, reason }] },
});

const appleOnly = {
  ...createMockConfig().mdm,
  enabled_and_configured: true,
  windows_enabled_and_configured: false,
};

const MANUAL_LOCATION = {
  pathname: "/controls/os-settings/configuration-profiles/new/manual",
  query: { fleet_id: 2 },
};

const renderGenerator = ({
  isPremiumTier = true,
  teamId = 2,
  mdm = appleOnly,
} = {}) => {
  const router = createMockRouter();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier,
        isGlobalAdmin: true,
        currentUser: createMockUser(),
        config: createMockConfig({ mdm }),
      },
    },
  });
  const results = render(
    <AIProfileGenerator
      router={router}
      teamId={teamId}
      teamName="Workstations"
      isPremiumTier={isPremiumTier}
      gitOpsModeEnabled={false}
      listPath="/controls/os-settings/configuration-profiles?fleet_id=2"
      manualLocation={MANUAL_LOCATION}
    />
  );
  return { ...results, router };
};

const typeInstructions = (value: string) =>
  fireEvent.change(screen.getByRole("textbox", { name: "Instructions" }), {
    target: { value },
  });

const clickGenerate = () =>
  fireEvent.click(screen.getByRole("button", { name: "Generate profile" }));

describe("AIProfileGenerator", () => {
  let nameDescription: IDeferred<IAutofillConfigProfileNameDescriptionResponse>;
  let content: IDeferred<IAutofillConfigProfileContentResponse>;

  beforeEach(() => {
    nameDescription = deferred();
    content = deferred();
    jest
      .spyOn(autofillAPI, "getConfigProfileNameDescription")
      .mockImplementation(() => nameDescription.promise);
    jest
      .spyOn(autofillAPI, "getConfigProfileContent")
      .mockImplementation(() => content.promise);
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: [] });
    jest
      .spyOn(mdmAPI, "uploadProfile")
      .mockResolvedValue({ profile_uuid: "a-new" });
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  it("offers only the platforms whose MDM is on", async () => {
    const { user } = renderGenerator();
    expect(screen.getByText("Apple (macOS, iOS, iPadOS)")).toBeInTheDocument();
    await user.click(screen.getByRole("combobox"));
    expect(screen.queryByText("Windows")).not.toBeInTheDocument();
  });

  it("names the fleet the profile is added to", () => {
    renderGenerator();
    expect(screen.getByText("Workstations")).toBeInTheDocument();
  });

  it("keeps Generate disabled until there are instructions, and past 2,000 characters", async () => {
    const { user } = renderGenerator();
    const generate = screen.getByRole("button", { name: "Generate profile" });
    expect(generate).toBeDisabled();
    await user.hover(generate);
    expect(
      await screen.findByText("Write instructions to generate a profile")
    ).toBeInTheDocument();

    typeInstructions("Turn off AirDrop");
    expect(generate).toBeEnabled();

    // counted in characters, as the API counts them
    typeInstructions("😀".repeat(2000));
    expect(generate).toBeEnabled();
    typeInstructions("😀".repeat(2001));
    expect(generate).toBeDisabled();
    await user.hover(generate);
    expect(
      await screen.findByText("You have exceeded the 2000 character limit.")
    ).toBeInTheDocument();
    expect(autofillAPI.getConfigProfileContent).not.toHaveBeenCalled();
  });

  it("sends both requests with the fleet, trimmed, and shows the progress", async () => {
    jest.useFakeTimers();
    renderGenerator();
    typeInstructions("  Turn off AirDrop  ");
    clickGenerate();

    expect(autofillAPI.getConfigProfileNameDescription).toHaveBeenCalledWith(
      expect.objectContaining({
        instructions: "Turn off AirDrop",
        platform: "apple",
      })
    );
    expect(autofillAPI.getConfigProfileContent).toHaveBeenCalledWith(
      expect.objectContaining({
        instructions: "Turn off AirDrop",
        platform: "apple",
        fleetId: 2,
      })
    );
    expect(screen.getByText("Reading request")).toBeInTheDocument();
    expect(screen.getByTestId("name-placeholder")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();

    act(() => jest.advanceTimersByTime(READING_STEP_MS));
    expect(screen.getByText("Drafting name and summary")).toBeInTheDocument();

    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    expect(
      screen.getByText("Writing and validating profile")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Disable AirDrop on Macs" })
    ).toBeInTheDocument();
    expect(screen.getByTestId("editor-placeholder")).toBeInTheDocument();
  });

  it.each([
    ["Unassigned", { teamId: 0 }],
    ["Free", { isPremiumTier: false }],
  ])("leaves the fleet out on %s", (_, options) => {
    renderGenerator(options);
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    expect(
      jest.mocked(autofillAPI.getConfigProfileContent).mock.calls[0][0].fleetId
    ).toBeUndefined();
  });

  it("shows a .mobileconfig that arrives first with its own name, and fills only the summary later", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();

    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));
    expect(
      screen.getByRole("heading", { name: "AirDrop Disabled" })
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Profile contents")).toHaveValue(MOBILECONFIG);
    expect(screen.getByTestId("description-placeholder")).toBeInTheDocument();
    expect(
      screen.getByText(
        "AI can make mistakes. Please double-check before you proceed."
      )
    ).toBeInTheDocument();

    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    expect(
      screen.getByRole("heading", { name: "AirDrop Disabled" })
    ).toBeInTheDocument();
    expect(screen.getByText("allowAirDrop")).toBeInTheDocument();
  });

  it("replaces the generated title with the .mobileconfig's name unless it was edited", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));

    fireEvent.click(screen.getByRole("button", { name: "Edit name" }));
    const field = screen.getByRole("textbox", { name: "Name" });
    fireEvent.change(field, { target: { value: "My AirDrop profile" } });
    fireEvent.keyDown(field, { key: "Enter" });

    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));
    expect(
      screen.getByRole("heading", { name: "My AirDrop profile" })
    ).toBeInTheDocument();
  });

  it("uses the generated title for a declaration, which has no name", async () => {
    renderGenerator();
    typeInstructions("Require a passcode");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    await act(async () => content.resolve(DECLARATION_RESPONSE));
    expect(
      screen.getByRole("heading", { name: "Disable AirDrop on Macs" })
    ).toBeInTheDocument();
    expect(screen.getByText("Declaration (DDM)")).toBeInTheDocument();
  });

  it("leaves a declaration's fields empty when the name request fails", async () => {
    renderGenerator();
    typeInstructions("Require a passcode");
    clickGenerate();
    await act(async () => nameDescription.reject(apiError("base", "x")));
    await act(async () => content.resolve(DECLARATION_RESPONSE));
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveValue("");
    expect(screen.getByRole("textbox", { name: "Description" })).toHaveValue(
      ""
    );
  });

  it("keeps a .mobileconfig's own name when the name request fails", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => nameDescription.reject(apiError("base", "x")));
    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));
    expect(
      screen.getByRole("heading", { name: "AirDrop Disabled" })
    ).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Description" })).toHaveValue(
      ""
    );
  });

  it("adds the reviewed profile with the generated name, description and type", async () => {
    const { user } = renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));

    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    const args = jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0];
    expect(args.name).toBe("AirDrop Disabled");
    expect(args.description).toBe("Sets `allowAirDrop` to `false`.");
    expect(args.teamId).toBe(2);
    expect(args.file.name).toMatch(/\.mobileconfig$/);
  });

  it("adds a generated profile the contents alone wouldn't type, going by the API's type", async () => {
    const { user } = renderGenerator();
    typeInstructions("Require a passcode");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    await act(async () =>
      content.resolve({
        ...DECLARATION_RESPONSE,
        // a secret placeholder makes it unparseable JSON
        content: `{"Type": "com.apple.configuration.passcode.settings", $FLEET_VAR_X}`,
      })
    );
    await user.click(screen.getByRole("button", { name: "Add profile" }));
    await waitFor(() => expect(mdmAPI.uploadProfile).toHaveBeenCalledTimes(1));
    expect(
      jest.mocked(mdmAPI.uploadProfile).mock.calls[0][0].file.name
    ).toMatch(/\.json$/);
  });

  it("keeps a title still being edited when the profile arrives", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));

    fireEvent.click(screen.getByRole("button", { name: "Edit name" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Name" }), {
      target: { value: "My AirDrop profile" },
    });
    // no blur or Enter before the profile replaces the view
    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));
    expect(
      screen.getByRole("heading", { name: "My AirDrop profile" })
    ).toBeInTheDocument();
  });

  it("offers Create manually with the result, and disables Retry while saving", async () => {
    const { user, router } = renderGenerator();
    const upload = deferred<{ profile_uuid: string }>();
    jest.mocked(mdmAPI.uploadProfile).mockImplementation(() => upload.promise);
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));

    expect(screen.getByRole("button", { name: "Retry" })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Add profile" }));
    expect(screen.getByRole("button", { name: "Retry" })).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "Create manually" }));
    expect(router.push).toHaveBeenCalledWith({
      ...MANUAL_LOCATION,
      state: { aiTimedOut: false },
    });
    await act(async () => upload.resolve({ profile_uuid: "a-new" }));
  });

  it("shows a decline with its reason and a link, and keeps the instructions", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => nameDescription.resolve(NAME_DESCRIPTION));
    await act(async () =>
      content.reject(
        apiError("natural_language_instructions", "AirDrop is a macOS feature.")
      )
    );
    expect(
      screen.getByText(
        /Your request could not be processed. AirDrop is a macOS feature./
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Learn more about OS settings/ })
    ).toHaveAttribute("href", "https://fleetdm.com/os-settings");
    expect(screen.getByRole("textbox", { name: "Instructions" })).toHaveValue(
      "Turn off AirDrop"
    );
    // the name shown while generating is discarded
    expect(
      screen.queryByText("Disable AirDrop on Macs")
    ).not.toBeInTheDocument();
  });

  it("shows any other failure in the generic banner", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () =>
      content.reject(apiError("base", "The AI service timed out."))
    );
    expect(
      screen.getByText(
        "Couldn't generate configuration profile. The AI service timed out. Please try again, or create your profile manually."
      )
    ).toBeInTheDocument();
  });

  it("stops both requests and returns to the instructions", () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    const { signal } = jest.mocked(
      autofillAPI.getConfigProfileContent
    ).mock.calls[0][0];
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(signal?.aborted).toBe(true);
    expect(
      screen.getByRole("button", { name: "Generate profile" })
    ).toBeEnabled();
  });

  it("goes to the manual view with the timeout banner at the cap", () => {
    jest.useFakeTimers();
    const { router } = renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    act(() => jest.advanceTimersByTime(GENERATION_TIMEOUT_MS));
    expect(router.push).toHaveBeenCalledWith({
      ...MANUAL_LOCATION,
      state: { aiTimedOut: true },
    });
  });

  it("stops the requests and goes to the manual view on Create manually, even while generating", () => {
    const { router } = renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    const { signal } = jest.mocked(
      autofillAPI.getConfigProfileContent
    ).mock.calls[0][0];
    fireEvent.click(screen.getByRole("button", { name: "Create manually" }));
    expect(signal?.aborted).toBe(true);
    expect(router.push).toHaveBeenCalledWith({
      ...MANUAL_LOCATION,
      state: { aiTimedOut: false },
    });
  });

  it("reruns both requests on Retry with the edited instructions", async () => {
    renderGenerator();
    typeInstructions("Turn off AirDrop");
    clickGenerate();
    await act(async () => content.resolve(MOBILECONFIG_RESPONSE));

    const retryContent = deferred<IAutofillConfigProfileContentResponse>();
    jest
      .mocked(autofillAPI.getConfigProfileContent)
      .mockImplementation(() => retryContent.promise);
    typeInstructions("Turn off AirDrop and Handoff");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(autofillAPI.getConfigProfileNameDescription).toHaveBeenCalledTimes(
      2
    );
    expect(
      jest.mocked(autofillAPI.getConfigProfileContent).mock.calls[1][0]
        .instructions
    ).toBe("Turn off AirDrop and Handoff");
    expect(screen.getByText("Reading request")).toBeInTheDocument();
  });
});
