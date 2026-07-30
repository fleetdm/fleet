import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import {
  createMockSoftwareTitle,
  createMockSoftwarePackage,
} from "__mocks__/softwareMock";
import { ISoftwarePackage } from "interfaces/software";
import softwareAPI from "services/entities/software";

import { notify } from "components/ToastNotification";

import VersionsModal from "./VersionsModal";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const fmaPackage = (overrides?: Partial<ISoftwarePackage>) =>
  createMockSoftwarePackage({
    fleet_maintained_app_id: 5,
    version: "149.0.2",
    fleet_maintained_versions: [
      {
        id: 1,
        version: "149.0.2",
        filename: "installer-149.0.2.pkg",
        uploaded_at: "2026-01-02T00:00:00Z",
      },
      {
        id: 2,
        version: "148.0.1",
        filename: "installer-148.0.1.pkg",
        uploaded_at: "2026-01-01T00:00:00Z",
      },
    ],
    ...overrides,
  });

const renderModal = (pkgOverrides?: Partial<ISoftwarePackage>) => {
  const onExit = jest.fn();
  const refetchSoftwareTitle = jest.fn();
  const render = createCustomRenderer({
    context: {
      app: {
        isPremiumTier: true,
        config: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          gitops: { gitops_mode_enabled: false, repository_url: "" } as any,
        },
      },
    },
  });
  const title = createMockSoftwareTitle({
    software_package: fmaPackage(pkgOverrides),
  });
  const utils = render(
    <VersionsModal
      softwareTitle={title}
      softwareId={1}
      teamId={1}
      refetchSoftwareTitle={refetchSoftwareTitle}
      onExit={onExit}
    />
  );
  return { ...utils, onExit, refetchSoftwareTitle };
};

describe("VersionsModal", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("preselects 'Automatically update to latest' when there is no pin, with Save disabled", () => {
    renderModal({ pinned_version: null });
    expect(
      screen.getByRole("radio", { name: "Automatically update to latest" })
    ).toBeChecked();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });

  it("preselects the matching exact-version radio for an exact pin", () => {
    renderModal({ pinned_version: "148.0.1" });
    expect(screen.getByRole("radio", { name: "Pin to 148.0.1" })).toBeChecked();
  });

  it("preselects the major-version radio for a caret pin", () => {
    renderModal({
      pinned_version: "^149",
      fleet_maintained_versions: [
        {
          id: 1,
          version: "149.0.2",
          filename: "installer-149.0.2.pkg",
          uploaded_at: "2026-01-02T00:00:00Z",
        },
        {
          id: 2,
          version: "149.0.1",
          filename: "installer-149.0.1.pkg",
          uploaded_at: "2026-01-01T00:00:00Z",
        },
      ],
    });
    expect(
      screen.getByRole("radio", { name: "Pin to major version (149)" })
    ).toBeChecked();
  });

  it("formats an aged-out caret pin as a major-version option (no '^' leak)", () => {
    // Pinned to ^148 but only 149.x is cached, so deriveVersionOptions emits
    // only ^149 — the fallback must synthesize a properly-labeled ^148 option.
    renderModal({
      pinned_version: "^148",
      fleet_maintained_versions: [
        {
          id: 1,
          version: "149.0.2",
          filename: "installer-149.0.2.pkg",
          uploaded_at: "2026-01-02T00:00:00Z",
        },
      ],
    });
    expect(
      screen.getByRole("radio", { name: "Pin to major version (148)" })
    ).toBeChecked();
    expect(screen.queryByText(/\^148/)).not.toBeInTheDocument();
  });

  it("enables Save once the selection changes and PATCHes the chosen version on save", async () => {
    const editSpy = jest
      .spyOn(softwareAPI, "editSoftwarePackage")
      .mockResolvedValue({} as never);
    const { user, onExit, refetchSoftwareTitle } = renderModal({
      pinned_version: null,
    });

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();

    await user.click(screen.getByRole("radio", { name: "Pin to 148.0.1" }));
    expect(saveButton).toBeEnabled();

    await user.click(saveButton);

    await waitFor(() => {
      expect(editSpy).toHaveBeenCalledWith({
        data: { pinnedVersion: "148.0.1" },
        softwareId: 1,
        teamId: 1,
      });
    });
    expect(notify.success).toHaveBeenCalledWith(expect.anything());
    expect(refetchSoftwareTitle).toHaveBeenCalled();
    expect(onExit).toHaveBeenCalled();
  });

  it("flashes an error and leaves the modal open when the PATCH fails", async () => {
    jest
      .spyOn(softwareAPI, "editSoftwarePackage")
      .mockRejectedValue(new Error("boom"));
    const { user, onExit } = renderModal({ pinned_version: null });

    await user.click(screen.getByRole("radio", { name: "Pin to 149.0.2" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        expect.stringContaining("Couldn't update version")
      );
    });
    expect(onExit).not.toHaveBeenCalled();
  });
});
