import React from "react";
import { render, screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import {
  MacMdmProfileOperationType,
  MacMdmProfileStatus,
} from "interfaces/mdm";
import MacSettingStatusCell from "./MacSettingStatusCell";

describe("Mac setting status cell", () => {
  it("Correctly displays the status text of a profile", () => {
    const status: MacMdmProfileStatus = "applied";
    const operationType: MacMdmProfileOperationType = "install";

    render(
      <MacSettingStatusCell status={status} operationType={operationType} />
    );

    expect(screen.getByText("Applied")).toBeInTheDocument();
  });

  it("Correctly displays the tooltip text for a profile", async () => {
    const status: MacMdmProfileStatus = "applied";
    const operationType: MacMdmProfileOperationType = "install";

    const customRender = createCustomRenderer();

    const { user } = customRender(
      <MacSettingStatusCell status={status} operationType={operationType} />
    );

    const statusText = screen.getByText("Applied");

    await user.hover(statusText);

    expect(screen.getByText("Host applied the setting.")).toBeInTheDocument();
  });
});
