import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockAppStoreAppVersion } from "__mocks__/softwareMock";
import softwareAPI from "services/entities/software";
import { createCustomRenderer } from "test/test-utils";

import EditVersionModal from "./EditVersionModal";

const mockVersion = createMockAppStoreAppVersion;

const BASE_PROPS = {
  softwareId: 42,
  teamId: 1,
  version: mockVersion({
    id: 41,
    app_store_id: "6443476492",
    version: "2.6.0",
    status: { installed: 10, pending: 0, failed: 0 },
    display_name: "Cloudflare One Agent",
    configuration: "<dict><key>SSO</key><true/></dict>",
    created_at: "2026-01-28T21:49:04.145909Z",
  }),
  titleDisplayName: "Cloudflare One Agent",
  siblingVersionNames: ["Test"],
  onExit: jest.fn(),
  onSuccess: jest.fn(),
};

const renderModal = (
  overrides: Partial<React.ComponentProps<typeof EditVersionModal>> = {}
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
  return render(<EditVersionModal {...BASE_PROPS} {...overrides} />);
};

describe("EditVersionModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders with the 'Edit version' title", () => {
    const { container } = renderModal();
    const header = container.querySelector(".modal__header");
    expect(header).toHaveTextContent("Edit version");
  });

  it("pre-fills Name from the version", () => {
    renderModal();
    expect(screen.getByLabelText(/Name/i)).toHaveValue("Production");
  });

  it("renders the Configuration editor", () => {
    renderModal();
    // Editor label renders synchronously; the ACE content area populates
    // async after mount so we only assert the label is present here.
    expect(screen.getByText(/Configuration/)).toBeInTheDocument();
  });

  it("surfaces duplicate-name error when renaming to a sibling name (case-insensitive)", async () => {
    const { user } = renderModal();
    const name = screen.getByLabelText(/Name/i);
    await user.clear(name);
    await user.type(name, "test");
    await user.tab();
    expect(
      await screen.findByText(
        /A version with this name already exists on this fleet/i
      )
    ).toBeInTheDocument();
  });

  it("pre-fills the Custom target when the version has labels_include_any", () => {
    renderModal({
      version: mockVersion({
        labels_include_any: [{ id: 1, name: "Production" }],
      }),
    });
    // The Custom radio should be selected (vs All hosts).
    expect(screen.getByLabelText(/Custom/i)).toBeChecked();
  });

  it("shows the auto-update checkbox checked when the version has auto_update_enabled", () => {
    renderModal({
      version: mockVersion({
        auto_update_enabled: true,
        auto_update_window_start: "00:00",
        auto_update_window_end: "04:00",
      }),
    });
    const checkbox = screen.getByLabelText(/Enable auto updates/i);
    expect(checkbox).toBeChecked();
  });

  // Android clear = {} (not undefined, which backend reads as "no change"). iOS/iPadOS clear = null.
  it("submits `{}` for Android when the editor holds the empty scaffold", async () => {
    const editSpy = jest
      .spyOn(softwareAPI, "editAppStoreAppVersion")
      .mockResolvedValue({} as never);

    const { user } = renderModal({
      version: mockVersion({
        platform: "android",
        app_store_id: "com.example.app",
        configuration: undefined,
      }),
    });

    await user.click(screen.getByRole("button", { name: /^Save$/i }));

    await waitFor(() => expect(editSpy).toHaveBeenCalled());
    const [, , , body] = editSpy.mock.calls[0];
    expect(body.configuration).toEqual({});
  });
});
