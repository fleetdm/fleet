import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import EnrollmentAttemptDetailsModal from "./EnrollmentAttemptDetailsModal";

const renderModal = (
  props: Partial<React.ComponentProps<typeof EnrollmentAttemptDetailsModal>>
) => {
  const render = createCustomRenderer();
  return render(<EnrollmentAttemptDetailsModal onDone={noop} {...props} />);
};

describe("EnrollmentAttemptDetailsModal", () => {
  it("names the host and explains a spent one-time secret", () => {
    renderModal({
      hostDisplayName: "Anna's MacBook Pro",
      reason: "one_time_secret_spent",
      createdAt: "2026-01-01T00:00:00Z",
    });

    expect(screen.getByText("Enrollment details")).toBeInTheDocument();
    expect(screen.getByText("Anna's MacBook Pro")).toBeInTheDocument();
    expect(
      screen.getByText(/one-time enroll secret was already used\. Resend the/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Fleetd configuration")).toBeInTheDocument();
    expect(screen.getByText("Host details > Controls")).toBeInTheDocument();
  });

  it.each([
    { platform: "darwin", profile: "Fleetd configuration" },
    { platform: "windows", profile: "Fleetd enroll secret" },
  ])(
    "names the $profile profile for a spent secret on $platform",
    ({ platform, profile }) => {
      renderModal({
        hostDisplayName: "Anna's laptop",
        reason: "one_time_secret_spent",
        platform,
      });
      expect(screen.getByText(profile)).toBeInTheDocument();
      const otherProfile =
        profile === "Fleetd configuration"
          ? "Fleetd enroll secret"
          : "Fleetd configuration";
      expect(screen.queryByText(otherProfile)).not.toBeInTheDocument();
    }
  );

  it("describes an identifier mismatch in the headline with a support link", () => {
    renderModal({
      hostDisplayName: "Anna's MacBook Pro",
      reason: "one_time_secret_identifier_mismatch",
    });
    expect(
      screen.getByText(
        /rejected an enrollment for a host that tried to enroll with/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Anna's MacBook Pro's")).toBeInTheDocument();
    expect(screen.getByText(/reach out to/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /fleet support/i })
    ).toHaveAttribute("href", "https://fleetdm.com/support");
    expect(
      screen.queryByText(/not valid for this host/i)
    ).not.toBeInTheDocument();
  });

  it("explains a shared secret used for a host that needs a one-time secret", () => {
    renderModal({ reason: "shared_secret_for_mdm_managed_host" });
    expect(
      screen.getByText(
        /A shared enroll secret was used for a host that requires a one-time enroll secret\./
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /how to troubleshoot/i })
    ).toHaveAttribute(
      "href",
      expect.stringMatching(/learn-more-about\/enrollment-troubleshooting$/)
    );
  });

  it("explains an automatic enrollment rejected because IdP authentication is required", () => {
    renderModal({
      hostDisplayName: "Anna's MacBook Pro",
      reason: "end_user_authentication_required",
    });
    expect(
      screen.getByText(/Fleet rejected an automatic enrollment for/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/requires IdP authentication, but the host tried/i)
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/rejected-automatic-enrollment"
    );
  });

  it("falls back to the serial number when there is no display name", () => {
    renderModal({ hostSerial: "C02ABC", reason: "one_time_secret_spent" });
    expect(screen.getByText(/a host with serial number/i)).toBeInTheDocument();
    expect(screen.getByText("C02ABC")).toBeInTheDocument();
  });

  it("uses the serial number in the identifier mismatch headline", () => {
    renderModal({
      hostSerial: "C02ABC",
      reason: "one_time_secret_identifier_mismatch",
    });
    expect(
      screen.getByText(
        /one-time enroll secret issued to a host with serial number/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText("C02ABC")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /fleet support/i })
    ).toBeInTheDocument();
  });

  it("falls back to a generic host and reason", () => {
    renderModal({ reason: "something_new" });

    expect(
      screen.getByText(/rejected an enrollment for a host\./i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/was not valid for this host/i)
    ).toBeInTheDocument();
  });

  it("calls onDone when closed", async () => {
    const onDone = jest.fn();
    const { user } = renderModal({ onDone });

    await user.click(screen.getByRole("button", { name: "Close" }));

    expect(onDone).toHaveBeenCalledTimes(1);
  });
});
