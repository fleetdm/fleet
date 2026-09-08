import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import { MDM_ENROLLMENT_TYPE_ACCOUNT_DRIVEN } from "interfaces/mdm";

import UnenrollMdmModal from "./UnenrollMdmModal";

const MOCK_PROPS = {
  hostId: 7,
  hostPlatform: "ios",
  hostName: "iphone-1",
  onlyAllowAppleBusinessEnrollment: false,
  depAssignedToFleet: false,
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

  describe("when onlyAllowAppleBusinessEnrollment is true", () => {
    describe("and depAssignedToFleet is false", () => {
      it.each(["ios", "ipados", "macos"])(
        "%s shows will not be able to enroll",
        (platform) => {
          const render = createCustomRenderer({ withBackendMock: true });
          const { rerender } = render(
            <UnenrollMdmModal
              {...MOCK_PROPS}
              hostPlatform={platform}
              onlyAllowAppleBusinessEnrollment
              depAssignedToFleet={false}
              enrollmentStatus="On (manual)"
              lastMdmEnrollmentType="Device"
            />
          );

          expect(
            screen.getByText(
              /Once MDM is turned off, this host will not be able to re-enroll/i
            )
          ).toBeInTheDocument();

          rerender(
            <UnenrollMdmModal
              {...MOCK_PROPS}
              hostPlatform={platform}
              onlyAllowAppleBusinessEnrollment
              depAssignedToFleet={false}
              enrollmentStatus="On (company-owned)"
              lastMdmEnrollmentType="Device"
            />
          );

          expect(
            screen.getByText(
              /Once MDM is turned off, this host will not be able to re-enroll/i
            )
          ).toBeInTheDocument();
        }
      );
    });
    describe("and depAssignedToFleet is true", () => {
      it.each(["ios", "ipados", "macos"])(
        "%s shows will not be able to enroll with current manual enrollment",
        (platform) => {
          const render = createCustomRenderer({ withBackendMock: true });
          render(
            <UnenrollMdmModal
              {...MOCK_PROPS}
              hostPlatform={platform}
              onlyAllowAppleBusinessEnrollment
              depAssignedToFleet
              enrollmentStatus="On (manual)"
              lastMdmEnrollmentType="Device"
            />
          );

          expect(
            screen.getByText(
              /Once MDM is turned off, this host will not be able to re-enroll/i
            )
          ).toBeInTheDocument();
        }
      );
      it.each(["ios", "ipados", "macos"])(
        "%s shows will be able to enroll with current automatic enrollment",
        (platform) => {
          const render = createCustomRenderer({ withBackendMock: true });
          render(
            <UnenrollMdmModal
              {...MOCK_PROPS}
              hostPlatform={platform}
              onlyAllowAppleBusinessEnrollment
              depAssignedToFleet
              enrollmentStatus="On (automatic)"
              lastMdmEnrollmentType="Device"
            />
          );

          expect(
            screen.getByText(
              /Once MDM is turned off, this host will be able to re-enroll/i
            )
          ).toBeInTheDocument();
        }
      );
    });
  });
});
