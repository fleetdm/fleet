import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import OktaConditionalAccessPage from "./OktaConditionalAccessPage";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const TEST_OKTA_IDP_ID = "okta-idp-123";
const TEST_OKTA_CERTIFICATE = "cert-data";
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

const createOktaConfig = (oktaConfigured = true) =>
  createMockConfig({
    conditional_access: {
      microsoft_entra_tenant_id: "",
      microsoft_entra_connection_configured: false,
      okta_idp_id: oktaConfigured ? TEST_OKTA_IDP_ID : "",
      okta_assertion_consumer_service_url: oktaConfigured
        ? "https://example.com/acs"
        : "",
      okta_audience_uri: oktaConfigured ? "https://example.com" : "",
      okta_certificate: oktaConfigured ? TEST_OKTA_CERTIFICATE : "",
      bypass_disabled: false,
    },
  });

const renderPage = (config = createOktaConfig()) => {
  const router = createMockRouter();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier: true, config } },
  });
  const utils = render(<OktaConditionalAccessPage router={router} />);
  return { ...utils, router };
};

// Ace renders text virtually, so the profiles are checked through each
// Editor's copy button, which copies the displayed value and is disabled
// while it's empty.
const getCopyButton = (ariaLabel: string) =>
  screen.getByRole("button", { name: `Copy ${ariaLabel}` });

describe("OktaConditionalAccessPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("shows the macOS profile by default and doesn't load the Windows profiles", async () => {
    mockServer.use(appleProfileHandler, windowsProfilesErrorHandler);

    const { user } = renderPage();

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

    const { user } = renderPage();

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    await waitFor(() => {
      expect(getCopyButton("certificate profile")).toBeEnabled();
    });
    expect(getCopyButton("browser profile")).toBeEnabled();
    expect(screen.queryByText("Profile")).not.toBeInTheDocument();

    // Spy after render: user-event's setup() replaces navigator.clipboard
    const writeTextSpy = jest.spyOn(navigator.clipboard, "writeText");
    await user.click(getCopyButton("certificate profile"));
    await user.click(getCopyButton("browser profile"));
    expect(writeTextSpy.mock.calls).toEqual([
      [TEST_WINDOWS_SCEP_PROFILE],
      [TEST_WINDOWS_BROWSER_PROFILE],
    ]);
  });

  it("shows the Windows error reason when the Windows profiles fail to load", async () => {
    mockServer.use(appleProfileHandler, windowsProfilesErrorHandler);

    const { user } = renderPage();

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Failed to load Windows profiles: global enroll secret is not configured",
        expect.anything()
      );
    });
    expect(getCopyButton("certificate profile")).toBeDisabled();
    expect(getCopyButton("browser profile")).toBeDisabled();
  });

  it("shows the Apple error reason when the macOS profile fails to load", async () => {
    mockServer.use(
      http.get(baseUrl("/conditional_access/idp/apple/profile"), () =>
        HttpResponse.json(
          {
            message: "Bad request",
            errors: [{ name: "base", reason: "missing private key" }],
          },
          { status: 400 }
        )
      )
    );

    renderPage();

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Failed to load Apple profile: missing private key",
        expect.anything()
      );
    });
    expect(getCopyButton("profile")).toBeDisabled();
  });

  it("opens the settings modal prefilled with the saved Okta settings", async () => {
    mockServer.use(appleProfileHandler);

    const { user } = renderPage();

    await user.click(screen.getByRole("button", { name: "Settings" }));

    expect(screen.getByLabelText("IdP ID")).toHaveValue(TEST_OKTA_IDP_ID);
    expect(screen.getByLabelText("Assertion consumer service URL")).toHaveValue(
      "https://example.com/acs"
    );
    expect(screen.getByLabelText("Audience URI")).toHaveValue(
      "https://example.com"
    );
  });

  it("saves edited settings without re-uploading the Okta certificate", async () => {
    let capturedBody: {
      conditional_access?: { okta_idp_id?: string; okta_certificate?: string };
    } | null = null;
    mockServer.use(
      appleProfileHandler,
      http.patch(baseUrl("/config"), async ({ request }) => {
        capturedBody = (await request.json()) as typeof capturedBody;
        return HttpResponse.json(createOktaConfig());
      })
    );

    const { user } = renderPage();

    await user.click(screen.getByRole("button", { name: "Settings" }));
    const idpIdInput = screen.getByLabelText("IdP ID");
    await user.clear(idpIdInput);
    await user.type(idpIdInput, "new-idp-id");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(capturedBody?.conditional_access).toEqual(
        expect.objectContaining({
          okta_idp_id: "new-idp-id",
          okta_certificate: TEST_OKTA_CERTIFICATE,
        })
      );
    });
    await waitFor(() => {
      expect(screen.queryByLabelText("IdP ID")).not.toBeInTheDocument();
    });
    expect(notify.success).toHaveBeenCalledWith(
      "Successfully updated Okta conditional access"
    );
  });

  it("goes back to conditional access when Okta isn't configured", async () => {
    const { router } = renderPage(createOktaConfig(false));

    await waitFor(() => {
      expect(router.replace).toHaveBeenCalledWith(
        "/settings/integrations/conditional-access"
      );
    });
  });
});
