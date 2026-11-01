import { screen } from "@testing-library/react";
import React from "react";

import { notify } from "components/ToastNotification";
import mdmAPI from "services/entities/mdm";
import { createCustomRenderer } from "test/test-utils";

import UnenrollMdmModal from "./UnenrollMdmModal";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));
jest.mock("services/entities/mdm");

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

  const conflictWithReason = (reason: string) => ({
    status: 409,
    response: { data: { errors: [{ name: "base", reason }] } },
  });

  it.each([
    {
      case: "android shows the server's reason",
      platform: "android",
      enrollmentStatus: "On (personal)" as const,
      conflict: conflictWithReason(
        "Couldn't unenroll. This host already has a pending unenroll. It will unenroll when it comes online."
      ),
      message:
        "Couldn't unenroll. This host already has a pending unenroll. It will unenroll when it comes online.",
    },
    {
      case: "ios says MDM is already off",
      platform: "ios",
      enrollmentStatus: "On (manual)" as const,
      conflict: conflictWithReason("ignored"),
      message: "Couldn't turn off MDM. This host already has MDM turned off.",
    },
  ])(
    "explains a 409 and refreshes the page: $case",
    async ({ platform, enrollmentStatus, conflict, message }) => {
      (mdmAPI.unenrollHostFromMdm as jest.Mock).mockRejectedValue(conflict);
      const render = createCustomRenderer();
      const { user } = render(
        <UnenrollMdmModal
          {...MOCK_PROPS}
          hostPlatform={platform}
          enrollmentStatus={enrollmentStatus}
        />
      );

      await user.click(screen.getByRole("button", { name: "Unenroll" }));

      expect(notify.error).toHaveBeenCalledWith(message, {
        response: conflict,
      });
      expect(MOCK_PROPS.onSuccess).toHaveBeenCalled();
      expect(MOCK_PROPS.onClose).toHaveBeenCalled();
    }
  );

  it("shows enrollment link instructions for a manual BYOD host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal
        {...MOCK_PROPS}
        enrollmentStatus="On (manual - personal)"
      />
    );

    expect(
      screen.getByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Sign in to Work or School Account/i)
    ).not.toBeInTheDocument();
  });

  it("shows sign-in instructions for an account-driven enrolled host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <UnenrollMdmModal {...MOCK_PROPS} enrollmentStatus="On (personal)" />
    );

    expect(
      screen.getByText(/Sign in to Work or School Account/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).not.toBeInTheDocument();
  });

  it("shows no BYOD re-enroll instructions for a company-owned manual host", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<UnenrollMdmModal {...MOCK_PROPS} enrollmentStatus="On (manual)" />);

    expect(
      screen.queryByText(/Hosts > Add hosts > iOS\/iPadOS/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Sign in to Work or School Account/i)
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
