import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import ClearPasscodeModal from "./ClearPasscodeModal";

describe("ClearPasscodeModal", () => {
  it("clears only the work profile passcode on an Android work-profile host", () => {
    render(
      <ClearPasscodeModal
        id={1}
        hostName="pixel"
        hostPlatform="android"
        hostMdmEnrollmentStatus="On (personal)"
        onExit={noop}
      />
    );

    expect(
      screen.getByText("This only clears the work profile passcode.")
    ).toBeInTheDocument();
  });

  it("clears the device passcode on a company-owned Android host", () => {
    render(
      <ClearPasscodeModal
        id={1}
        hostName="pixel"
        hostPlatform="android"
        hostMdmEnrollmentStatus="On (automatic)"
        onExit={noop}
      />
    );

    expect(
      screen.getByText(/This will clear the host passcode/)
    ).toBeInTheDocument();
    expect(
      screen.queryByText("This only clears the work profile passcode.")
    ).not.toBeInTheDocument();
  });
});
