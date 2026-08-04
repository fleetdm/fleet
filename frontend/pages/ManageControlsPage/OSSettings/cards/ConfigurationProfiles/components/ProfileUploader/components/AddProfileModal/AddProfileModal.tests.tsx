import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import labelsAPI from "services/entities/labels";
import mdmAPI from "services/entities/mdm";

import AddProfileModal from "./AddProfileModal";

const ADVANCED_OPTIONS = "Advanced options";

const DDM_DECLARATION = JSON.stringify({
  Type: "com.apple.configuration.management.test",
  Identifier: "com.fleetdm.config.test",
  Payload: { Echo: "test" },
});

const ANDROID_PROFILE = JSON.stringify({ screenCaptureDisabled: true });

const ddmFile = () =>
  new File([DDM_DECLARATION], "declaration.json", {
    type: "application/json",
  });

const androidFile = () =>
  new File([ANDROID_PROFILE], "android.json", { type: "application/json" });

const render = createCustomRenderer({ withBackendMock: true });

const renderModal = (isPremiumTier: boolean, setShowModal = noop) =>
  render(
    <AddProfileModal
      currentTeamId={0}
      isPremiumTier={isPremiumTier}
      onUpload={noop}
      setShowModal={setShowModal}
    />
  );

const getFileInput = (container: HTMLElement) =>
  container.querySelector('input[type="file"]') as HTMLInputElement;

describe("AddProfileModal advanced options", () => {
  beforeEach(() => {
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: [] });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("hides advanced options until a file is chosen", async () => {
    renderModal(true);

    await screen.findByText("Target");

    expect(screen.queryByText(ADVANCED_OPTIONS)).not.toBeInTheDocument();
  });

  it("shows advanced options for a DDM declaration on premium", async () => {
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(getFileInput(container), ddmFile());

    expect(await screen.findByText(ADVANCED_OPTIONS)).toBeInTheDocument();
  });

  it("hides advanced options for an Android profile", async () => {
    // same .json extension as a declaration -- only the contents differ, so
    // this proves the check isn't a file-extension sniff.
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(getFileInput(container), androidFile());

    await screen.findByText("android");
    expect(screen.queryByText(ADVANCED_OPTIONS)).not.toBeInTheDocument();
  });

  it("hides advanced options for a .mobileconfig profile", async () => {
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(
      getFileInput(container),
      new File(["<plist></plist>"], "profile.mobileconfig", {
        type: "application/x-apple-aspen-config",
      })
    );

    await screen.findByText("profile");
    expect(screen.queryByText(ADVANCED_OPTIONS)).not.toBeInTheDocument();
  });

  it("hides advanced options for a Windows profile", async () => {
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(
      getFileInput(container),
      new File(["<Replace></Replace>"], "windows.xml", { type: "text/xml" })
    );

    await screen.findByText("windows");
    expect(screen.queryByText(ADVANCED_OPTIONS)).not.toBeInTheDocument();
  });

  it("hides advanced options for a DDM declaration on free tier", async () => {
    // custom activations are premium-only.
    const { user, container } = renderModal(false);

    await user.upload(getFileInput(container), ddmFile());

    await screen.findByText("declaration");
    expect(screen.queryByText(ADVANCED_OPTIONS)).not.toBeInTheDocument();
  });

  it("offers no way to swap the file once one is chosen", async () => {
    // the modal replaces the file chooser with a read-only summary after a
    // selection, so a declaration can never be swapped for an Android profile
    // in place -- the admin has to cancel and reopen. This is why there is no
    // test for the section hiding again after a swap.
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(getFileInput(container), ddmFile());
    await screen.findByText(ADVANCED_OPTIONS);

    await waitFor(() => {
      expect(getFileInput(container)).not.toBeInTheDocument();
    });
  });

  it("reveals the activation editor for a declaration", async () => {
    // NOTE: the editor's contents can't be asserted here -- ace can't measure
    // text in jsdom and renders placeholder glyphs instead of the value. The
    // example itself is covered by the EXAMPLE_CUSTOM_ACTIVATION unit tests.
    const { user, container } = renderModal(true);

    await screen.findByText("Target");
    await user.upload(getFileInput(container), ddmFile());
    await user.click(
      await screen.findByRole("button", { name: ADVANCED_OPTIONS })
    );

    expect(screen.getByText("Custom activation")).toBeInTheDocument();
    expect(
      container.querySelector(".profile-advanced-options__custom-activation")
    ).toBeInTheDocument();
  });
});

describe("AddProfileModal upload result", () => {
  beforeEach(() => {
    jest.spyOn(labelsAPI, "summary").mockResolvedValue({ labels: [] });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  const uploadDeclaration = async (setShowModal: () => void) => {
    const { user, container } = renderModal(true, setShowModal);

    await screen.findByText("Target");
    await user.upload(getFileInput(container), ddmFile());
    await user.click(
      await screen.findByRole("button", { name: "Add profile" })
    );
  };

  it("keeps the modal open when the upload is rejected", async () => {
    const setShowModal = jest.fn();
    jest.spyOn(mdmAPI, "uploadProfile").mockRejectedValue({
      data: {
        errors: [
          { name: "base", reason: "The profile should include valid JSON" },
        ],
      },
    });

    await uploadDeclaration(setShowModal);

    await waitFor(() => {
      expect(mdmAPI.uploadProfile).toHaveBeenCalled();
    });
    expect(setShowModal).not.toHaveBeenCalled();
    // the chosen file survives the rejection so the admin can correct and retry
    expect(screen.getByText("declaration")).toBeInTheDocument();
  });

  it("closes the modal when the upload succeeds", async () => {
    const setShowModal = jest.fn();
    jest.spyOn(mdmAPI, "uploadProfile").mockResolvedValue({});

    await uploadDeclaration(setShowModal);

    await waitFor(() => {
      expect(setShowModal).toHaveBeenCalledWith(false);
    });
  });
});
