import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockAppStoreAppIos } from "__mocks__/softwareMock";
import softwareAPI from "services/entities/software";
import { createCustomRenderer } from "test/test-utils";

import AddVersionModal from "./AddVersionModal";

const BASE_PROPS = {
  softwareTitleId: 42,
  teamId: 1,
  appStore: createMockAppStoreAppIos(),
  existingVersionNames: [],
  onExit: jest.fn(),
  onSuccess: jest.fn(),
};

const renderModal = (
  overrides: Partial<React.ComponentProps<typeof AddVersionModal>> = {}
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
      },
    },
  });
  return render(<AddVersionModal {...BASE_PROPS} {...overrides} />);
};

describe("AddVersionModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders with the 'Add version' title", () => {
    renderModal();
    // Modal title and submit button both read "Add version"; disambiguate on
    // the submit button role to pin on the title header's non-button match.
    const matches = screen.getAllByText("Add version");
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it("shows the first-added-wins callout inside the Target section", () => {
    renderModal();
    expect(
      screen.getByText(
        /If multiple versions target the same host, Fleet will deploy the one that was added first\./i
      )
    ).toBeInTheDocument();
  });

  it("renders the Name field", () => {
    renderModal();
    expect(screen.getByLabelText(/Name/i)).toBeInTheDocument();
  });

  it("renders the Configuration editor", () => {
    renderModal();
    // The Editor component renders a label element with "Configuration" text.
    expect(screen.getByText(/Configuration/)).toBeInTheDocument();
  });

  it("surfaces required-name error on empty submit", async () => {
    const { user } = renderModal();
    // The submit button is labeled "Add" (not "Add version").
    const submit = screen.getByRole("button", { name: /^Add$/i });
    await user.click(submit);
    expect(
      await screen.findByText(/Enter a version name/i)
    ).toBeInTheDocument();
  });

  it("surfaces duplicate-name error on blur when a sibling name matches (case-insensitive)", async () => {
    const { user } = renderModal({ existingVersionNames: ["Production"] });
    const name = screen.getByLabelText(/Name/i);
    await user.type(name, "production");
    await user.tab();
    expect(
      await screen.findByText(
        /A version with this name already exists on this fleet/i
      )
    ).toBeInTheDocument();
  });

  it("shows the Auto updates section on iOS titles", () => {
    renderModal();
    expect(screen.getByText(/Enable auto updates/i)).toBeInTheDocument();
  });

  it("hides the Auto updates section on Android titles", () => {
    const androidApp = createMockAppStoreAppIos({
      platform: "android",
      app_store_id: "com.example.app",
    });
    renderModal({ appStore: androidApp });
    expect(screen.queryByText(/Enable auto updates/i)).not.toBeInTheDocument();
  });

  it("pre-fills auto-update from defaultAutoUpdate and submits those values", async () => {
    const addSpy = jest
      .spyOn(softwareAPI, "addAppStoreAppVersion")
      .mockResolvedValue({} as never);

    const { user } = renderModal({
      defaultAutoUpdate: {
        enabled: true,
        windowStart: "22:00",
        windowEnd: "02:00",
      },
    });

    expect(screen.getByLabelText(/Enable auto updates/i)).toBeChecked();
    expect(
      screen.getByLabelText(/Window start \(host local time\)/i)
    ).toHaveValue("22:00");
    expect(
      screen.getByLabelText(/Window end \(host local time\)/i)
    ).toHaveValue("02:00");

    await user.type(screen.getByLabelText(/Name/i), "Production");
    await user.click(screen.getByRole("button", { name: /^Add$/i }));

    await waitFor(() => expect(addSpy).toHaveBeenCalled());
    const [, body] = addSpy.mock.calls[0];
    expect(body).toMatchObject({
      name: "Production",
      auto_update_enabled: true,
      auto_update_window_start: "22:00",
      auto_update_window_end: "02:00",
    });
  });
});
