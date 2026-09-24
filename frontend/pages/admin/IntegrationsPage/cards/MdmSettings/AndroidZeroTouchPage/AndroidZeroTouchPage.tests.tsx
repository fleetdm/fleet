import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockAxiosError from "__mocks__/axiosError";
import createMockUser from "__mocks__/userMock";
import mdmAndroidAPI from "services/entities/mdm_android";
import { createCustomRenderer } from "test/test-utils";

import AndroidZeroTouchPage from "./AndroidZeroTouchPage";

const CONFIGURED_APP_CONTEXT = {
  currentUser: createMockUser(),
  isPremiumTier: true,
  isAndroidMdmEnabledAndConfigured: true,
};

describe("AndroidZeroTouchPage", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  // `isPremiumTier` is undefined until the config request resolves. Neither the
  // paywall nor the premium-only request may appear in that window.
  test("shows neither the paywall nor a request until the tier is known", async () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { currentUser: createMockUser(), isPremiumTier: undefined },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.queryByText(/Fleet Premium/i)).toBeNull();
    expect(getConfig).not.toHaveBeenCalled();
    expect(await screen.findByTestId("spinner")).toBeVisible();
  });

  test("shows premium upsell without calling the API when not premium tier", () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...CONFIGURED_APP_CONTEXT,
          isPremiumTier: false,
        },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Fleet Premium/i)).toBeVisible();
    expect(getConfig).not.toHaveBeenCalled();
  });

  test("shows the prerequisite message when Android MDM is off", () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...CONFIGURED_APP_CONTEXT,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.getByText(/first turn on Android MDM/i)).toBeVisible();
    expect(screen.queryByText("DPC extras")).toBeNull();
    expect(getConfig).not.toHaveBeenCalled();
  });

  test("shows DPC extras after loading", async () => {
    jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration").mockResolvedValue({
      dpc_extras: '{"test": "dpc-extras-json"}',
      expires_at: "2126-09-08T00:00:00Z",
    });

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    await waitFor(() => {
      expect(screen.getByText(/dpc-extras-json/)).toBeVisible();
    });

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Android zero-touch portal/)).toBeVisible();
    expect(screen.getByText(/Unassigned/)).toBeVisible();
    expect(screen.getByText(/Add configuration/)).toBeVisible();
    expect(screen.getByRole("button", { name: /copy/i })).toBeVisible();
  });

  test("shows error state on API failure and hides the copy button", async () => {
    jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockRejectedValue(createMockAxiosError({ status: 403 }));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    await waitFor(() => {
      expect(screen.getByText(/gone wrong/i)).toBeVisible();
    });

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText("DPC extras")).toBeVisible();
    expect(screen.queryByRole("button", { name: /copy/i })).toBeNull();
    expect(screen.queryByText(/Add configuration/)).toBeNull();
  });
});
