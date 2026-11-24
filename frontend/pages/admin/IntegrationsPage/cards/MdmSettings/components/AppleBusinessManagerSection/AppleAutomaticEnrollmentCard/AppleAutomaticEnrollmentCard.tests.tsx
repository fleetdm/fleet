import { render, screen } from "@testing-library/react";
import React from "react";

import AppleAutomaticEnrollmentCard from "./AppleAutomaticEnrollmentCard";

describe("AppleAutomaticEnrollmentCard", () => {
  const defaultProps = {
    isAppleMdmOn: true,
    viewDetails: jest.fn(),
    configured: false,
    onlyAllowAppleBusinessEnrollment: false,
  };

  it("shows a plain apostrophe, not a literal HTML entity, when personal enrollment is also allowed", () => {
    render(<AppleAutomaticEnrollmentCard {...defaultProps} />);

    expect(
      screen.getByText(/when they're first unboxed\. Personal \(BYOD\) hosts/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/&apos;/)).not.toBeInTheDocument();
  });

  it("shows a plain apostrophe when only Apple Business enrollment is allowed", () => {
    render(
      <AppleAutomaticEnrollmentCard
        {...defaultProps}
        onlyAllowAppleBusinessEnrollment
      />
    );

    expect(
      screen.getByText(/when they're first unboxed\./)
    ).toBeInTheDocument();
    expect(screen.queryByText(/&apos;/)).not.toBeInTheDocument();
  });
});
