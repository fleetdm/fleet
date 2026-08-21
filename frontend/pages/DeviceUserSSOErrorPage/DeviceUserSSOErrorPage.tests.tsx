import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";

import DeviceUserSSOErrorPage from "./DeviceUserSSOErrorPage";

const render = createCustomRenderer();

// Required for useIsMobileWidth
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: jest.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
      addListener: jest.fn(),
      removeListener: jest.fn(),
      onchange: null,
      dispatchEvent: jest.fn(),
    })),
  });
});

const renderPage = (reason?: string) =>
  render(
    <DeviceUserSSOErrorPage
      router={createMockRouter()}
      params={{}}
      routes={[]}
      location={{
        pathname: "/device/sso-error",
        search: "",
        query: reason ? { reason } : {},
        hash: "",
        action: "POP",
        key: "",
        state: undefined,
      }}
    />
  );

describe("DeviceUserSSOErrorPage", () => {
  it("tells the end user their sign-in session ran out", () => {
    renderPage("session_expired");

    expect(
      screen.getByText("Your sign-in session expired.")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Open Fleet Desktop and click/)
    ).toBeInTheDocument();
  });

  it("falls back to the generic failure for any other reason", () => {
    renderPage("error");

    expect(screen.getByText("Couldn't finish signing in.")).toBeInTheDocument();
  });

  it("does not leak a device token into the page", () => {
    // The route exists precisely because the callback cannot know the device
    // page URL at this point, so nothing here should link back to one.
    const { container } = renderPage();

    expect(container.querySelector("a")).toBeNull();
  });
});
