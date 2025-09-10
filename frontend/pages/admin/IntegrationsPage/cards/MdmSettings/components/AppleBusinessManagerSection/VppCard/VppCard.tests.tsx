import React from "react";
import noop from "lodash/noop";

import { render, screen } from "@testing-library/react";

import VppCard from "./VppCard";

describe("Vpp Card", () => {
  it("renders mdm is off message when apple mdm is not turned on ", async () => {
    render(<VppCard viewDetails={noop} isVppOn={false} isAppleMdmOn={false} />);

    expect(
      await screen.findByText(
        "To enable Volume Purchasing Program (VPP), first turn on Apple (macOS, iOS, iPadOS) MDM."
      )
    ).toBeInTheDocument();
  });

  it("renders add vpp when vpp is disabled", async () => {
    render(<VppCard viewDetails={noop} isVppOn={false} isAppleMdmOn />);

    expect(
      await screen.findByRole("button", { name: "Add VPP" })
    ).toBeInTheDocument();
  });

  it("renders edit vpp when vpp is enabled", async () => {
    render(<VppCard viewDetails={noop} isVppOn isAppleMdmOn />);
    expect(
      await screen.findByRole("button", { name: "Edit" })
    ).toBeInTheDocument();
  });
});
