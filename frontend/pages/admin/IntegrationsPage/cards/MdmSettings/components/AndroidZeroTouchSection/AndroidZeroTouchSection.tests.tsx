import { screen } from "@testing-library/react";
import React from "react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";

import AndroidZeroTouchSection from "./AndroidZeroTouchSection";

describe("AndroidZeroTouchSection", () => {
  test("shows premium upsell when not premium tier", () => {
    const render = createCustomRenderer({
      context: {
        app: { isAndroidMdmEnabledAndConfigured: true },
      },
    });

    render(
      <AndroidZeroTouchSection
        router={createMockRouter()}
        isPremiumTier={false}
      />
    );

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Fleet Premium/i)).toBeVisible();
  });

  test("shows disabled message when Android MDM is off", () => {
    const render = createCustomRenderer({
      context: {
        app: { isAndroidMdmEnabledAndConfigured: false },
      },
    });

    render(
      <AndroidZeroTouchSection router={createMockRouter()} isPremiumTier />
    );

    expect(screen.getByText("Android enrollment")).toBeVisible();
    expect(screen.getByText(/first turn on Android MDM/i)).toBeVisible();
  });

  test("shows setup button when Android MDM is on and premium", () => {
    const render = createCustomRenderer({
      context: {
        app: { isAndroidMdmEnabledAndConfigured: true },
      },
    });

    render(
      <AndroidZeroTouchSection router={createMockRouter()} isPremiumTier />
    );

    expect(
      screen.getByText(/automatically enroll company-owned/i)
    ).toBeVisible();
    expect(screen.getByText("Setup")).toBeVisible();
  });
});
