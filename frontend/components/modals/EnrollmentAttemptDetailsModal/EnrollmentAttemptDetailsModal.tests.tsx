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

  it("explains an identifier mismatch", () => {
    renderModal({ reason: "one_time_secret_identifier_mismatch" });
    expect(
      screen.getByText(/different serial number or hardware UUID/i)
    ).toBeInTheDocument();
  });

  it("explains a shared secret used for a host that needs a one-time secret", () => {
    renderModal({ reason: "shared_secret_for_mdm_managed_host" });
    expect(
      screen.getByText(
        "A shared enroll secret was used for a host that requires a one-time enroll secret."
      )
    ).toBeInTheDocument();
  });

  it("falls back to a generic host and reason", () => {
    renderModal({ reason: "something_new" });

    expect(screen.getByText("a host")).toBeInTheDocument();
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
