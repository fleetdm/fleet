import { screen } from "@testing-library/react";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

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

  it("shows enrollment link instructions for a manual BYOD host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<UnenrollMdmModal {...MOCK_PROPS} enrollmentStatus="On (manual)" />);

    expect(
      screen.getByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Sign in to Work or School Account/i)
    ).not.toBeInTheDocument();
  });

  it("shows sign-in instructions for a personally enrolled host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal
        {...MOCK_PROPS}
        enrollmentStatus="On (manual - personal)"
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
      <UnenrollMdmModal {...MOCK_PROPS} enrollmentStatus="On (automatic)" />
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
