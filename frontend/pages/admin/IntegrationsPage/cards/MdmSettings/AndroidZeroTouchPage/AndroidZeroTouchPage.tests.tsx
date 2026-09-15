import { screen, waitFor } from "@testing-library/react";
import React from "react";

import mdmAndroidAPI from "services/entities/mdm_android";
import { createCustomRenderer } from "test/test-utils";

import AndroidZeroTouchPage from "./AndroidZeroTouchPage";

describe("AndroidZeroTouchPage", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  test("shows premium upsell when not premium tier", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: false },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Fleet Premium/i)).toBeVisible();
  });

  test("shows DPC extras after loading", async () => {
    jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration").mockResolvedValue({
      dpc_extras: '{"test": "dpc-extras-json"}',
      expires_at: "2126-09-08T00:00:00Z",
    });

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true },
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
  });

  test("shows error state on API failure", async () => {
    jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockRejectedValue(new Error("API error"));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true },
      },
    });

    render(<AndroidZeroTouchPage />);

    await waitFor(() => {
      expect(screen.getByText(/gone wrong/i)).toBeVisible();
    });

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText("DPC extras")).toBeVisible();
  });
});
