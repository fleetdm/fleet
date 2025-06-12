import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import AndroidMdmCard from "./AndroidMdmCard";

describe("AndroidMdmCard", () => {
  test("render the expected content when Android MDM is turned off", () => {
    const render = createCustomRenderer({
      context: {
        app: { isAndroidMdmEnabledAndConfigured: false },
      },
    });

    render(<AndroidMdmCard turnOffAndroidMdm={noop} editAndroidMdm={noop} />);

    expect(screen.getByText("Turn on Android MDM")).toBeVisible();
    expect(screen.getByRole("button", { name: "Turn on" })).toBeVisible();
  });

  test("render the expected content when Android MDM is turned on", () => {
    const render = createCustomRenderer({
      context: {
        app: { isAndroidMdmEnabledAndConfigured: true },
      },
    });

    render(<AndroidMdmCard turnOffAndroidMdm={noop} editAndroidMdm={noop} />);

    expect(screen.getByText("Android MDM turned on.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Edit" })).toBeVisible();
  });
});
