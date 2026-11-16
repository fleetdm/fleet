import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import OktaProfilesModal from "./OktaProfilesModal";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const TEST_APPLE_PROFILE = "<plist>apple</plist>";
const TEST_WINDOWS_SCEP_PROFILE = "<Replace>windows scep</Replace>";
const TEST_WINDOWS_BROWSER_PROFILE = "<Replace>windows browser</Replace>";

const appleProfileHandler = http.get(
  baseUrl("/conditional_access/idp/apple/profile"),
  () => HttpResponse.text(TEST_APPLE_PROFILE)
);

const windowsProfilesHandler = http.get(
  baseUrl("/conditional_access/idp/windows/profiles"),
  () =>
    HttpResponse.json({
      scep_profile: TEST_WINDOWS_SCEP_PROFILE,
      browser_profile: TEST_WINDOWS_BROWSER_PROFILE,
    })
);

const windowsProfilesErrorHandler = http.get(
  baseUrl("/conditional_access/idp/windows/profiles"),
  () =>
    HttpResponse.json(
      {
        message: "Bad request",
        errors: [
          { name: "base", reason: "global enroll secret is not configured" },
        ],
      },
      { status: 400 }
    )
);

const renderModal = (onExit = jest.fn()) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier: true } },
  });
  return { ...render(<OktaProfilesModal onExit={onExit} />), onExit };
};

// Ace renders text virtually, so the profiles are checked through each
// Editor's copy button, which copies the displayed value and is disabled
// while it's empty.
const getCopyButton = (ariaLabel: string) =>
  screen.getByRole("button", { name: `Copy ${ariaLabel}` });

describe("OktaProfilesModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("shows the macOS profile by default and doesn't load the Windows profiles", async () => {
    // The Windows handler errors, so loading it would fire a toast.
    mockServer.use(appleProfileHandler, windowsProfilesErrorHandler);

    const { user } = renderModal();

    expect(
      screen.getByRole("tab", { name: "macOS", selected: true })
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(getCopyButton("profile")).toBeEnabled();
    });
    expect(screen.queryByText("Certificate profile")).not.toBeInTheDocument();

    const writeTextSpy = jest.spyOn(navigator.clipboard, "writeText");
    await user.click(getCopyButton("profile"));
    expect(writeTextSpy).toHaveBeenCalledWith(TEST_APPLE_PROFILE);
    expect(notify.error).not.toHaveBeenCalled();
  });

  it("shows both Windows profiles on the Windows tab and copies each one", async () => {
    mockServer.use(appleProfileHandler, windowsProfilesHandler);

    const { user } = renderModal();

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    await waitFor(() => {
      expect(getCopyButton("certificate profile")).toBeEnabled();
    });
    expect(getCopyButton("browser profile")).toBeEnabled();
    expect(
      screen.getByText(
        "Replace the placeholder with a global enroll secret before uploading. For GitOps, use a secret variable instead."
      )
    ).toBeInTheDocument();

    // Spy after render: user-event's setup() replaces navigator.clipboard
    const writeTextSpy = jest.spyOn(navigator.clipboard, "writeText");
    await user.click(getCopyButton("certificate profile"));
    await user.click(getCopyButton("browser profile"));
    expect(writeTextSpy.mock.calls).toEqual([
      [TEST_WINDOWS_SCEP_PROFILE],
      [TEST_WINDOWS_BROWSER_PROFILE],
    ]);
  });

  it("shows the server's reason when the Windows profiles fail to load", async () => {
    mockServer.use(appleProfileHandler, windowsProfilesErrorHandler);

    const { user } = renderModal();

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        expect.stringContaining("global enroll secret is not configured"),
        expect.anything()
      );
    });
    expect(getCopyButton("certificate profile")).toBeDisabled();
    expect(getCopyButton("browser profile")).toBeDisabled();
  });

  it("shows the server's reason when the macOS profile fails to load", async () => {
    mockServer.use(
      http.get(baseUrl("/conditional_access/idp/apple/profile"), () =>
        HttpResponse.json(
          {
            message: "Bad request",
            errors: [{ name: "base", reason: "missing SCEP challenge" }],
          },
          { status: 400 }
        )
      )
    );

    renderModal();

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        expect.stringContaining("missing SCEP challenge"),
        expect.anything()
      );
    });
    expect(getCopyButton("profile")).toBeDisabled();
  });

  it("calls onExit when Close is clicked", async () => {
    mockServer.use(appleProfileHandler);

    const { user, onExit } = renderModal();

    await user.click(screen.getByRole("button", { name: "Close" }));

    expect(onExit).toHaveBeenCalledTimes(1);
  });
});
