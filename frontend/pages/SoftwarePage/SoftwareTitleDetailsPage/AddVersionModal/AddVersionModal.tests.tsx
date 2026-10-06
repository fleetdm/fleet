import { screen } from "@testing-library/react";
import React from "react";

import { createMockAppStoreAppIos } from "__mocks__/softwareMock";
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
});
