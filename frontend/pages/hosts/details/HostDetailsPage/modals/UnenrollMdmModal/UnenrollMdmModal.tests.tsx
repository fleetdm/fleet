import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import { MDM_ENROLLMENT_TYPE_ACCOUNT_DRIVEN } from "interfaces/mdm";

import UnenrollMdmModal from "./UnenrollMdmModal";

const MOCK_PROPS = {
  hostId: 7,
  hostPlatform: "ios",
  hostName: "iphone-1",
  onSuccess: jest.fn(),
  onClose: jest.fn(),
};

describe("UnenrollMdmModal", () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  // Manual BYOD and account-driven hosts share the "On (manual - personal)"
  // status, so the status alone must not decide which instructions to show.
  // Following the account-driven steps on a manual BYOD device fails with
  // "Your Apple Account does not support the expected services". See #50868.
  it("shows enrollment link instructions for a manual BYOD host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal
        {...MOCK_PROPS}
        enrollmentStatus="On (manual - personal)"
        lastMdmEnrollmentType="Device"
      />
    );

    expect(
      screen.getByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Sign in to Work or School Account/i)
    ).not.toBeInTheDocument();
  });

  it("shows sign-in instructions for an account-driven host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal
        {...MOCK_PROPS}
        enrollmentStatus="On (manual - personal)"
        lastMdmEnrollmentType={MDM_ENROLLMENT_TYPE_ACCOUNT_DRIVEN}
      />
    );

    expect(
      screen.getByText(/Sign in to Work or School Account/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).not.toBeInTheDocument();
  });

  it("shows Apple Business instructions for an automatically enrolled host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal
        {...MOCK_PROPS}
        enrollmentStatus="On (automatic)"
        lastMdmEnrollmentType="Device"
      />
    );

    expect(
      screen.getByText(/make sure that the host is still in Apple Business/i)
    ).toBeInTheDocument();
  });
});
