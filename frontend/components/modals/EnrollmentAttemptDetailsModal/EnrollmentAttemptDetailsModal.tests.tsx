import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import EnrollmentAttemptDetailsModal, {
  getEnrollmentRejectedReasonText,
} from "./EnrollmentAttemptDetailsModal";

describe("EnrollmentAttemptDetailsModal", () => {
  it("names the host and explains the reason", () => {
    const render = createCustomRenderer();
    render(
      <EnrollmentAttemptDetailsModal
        hostDisplayName="Anna's MacBook Pro"
        reason="one_time_secret_spent"
        createdAt="2026-01-01T00:00:00Z"
        onDone={noop}
      />
    );

    expect(screen.getByText("Enrollment attempt details")).toBeInTheDocument();
    expect(screen.getByText("Anna's MacBook Pro")).toBeInTheDocument();
    expect(
      screen.getByText(
        /one-time enroll secret was already used\. Resend the "Fleetd configuration" profile/i
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /fleet support/i })
    ).toHaveAttribute("href", "https://fleetdm.com/support");
  });

  it("falls back to a generic host and reason", () => {
    const render = createCustomRenderer();
    render(<EnrollmentAttemptDetailsModal onDone={noop} />);

    expect(screen.getByText("a host")).toBeInTheDocument();
    expect(
      screen.getByText(/was not valid for this host/i)
    ).toBeInTheDocument();
  });

  it("calls onDone when closed", async () => {
    const onDone = jest.fn();
    const render = createCustomRenderer();
    const { user } = render(<EnrollmentAttemptDetailsModal onDone={onDone} />);

    await user.click(screen.getByRole("button", { name: "Close" }));

    expect(onDone).toHaveBeenCalledTimes(1);
  });
});

describe("getEnrollmentRejectedReasonText", () => {
  const cases: Array<[string | undefined, RegExp]> = [
    ["one_time_secret_spent", /already used/],
    [
      "one_time_secret_identifier_mismatch",
      /different serial number or hardware UUID/,
    ],
    [
      "shared_secret_for_mdm_managed_host",
      /global or fleet-level enroll secret/,
    ],
    [undefined, /not valid for this host/],
    ["something_new", /not valid for this host/],
  ];

  it.each(cases)("returns copy for reason %j", (reason, expected) => {
    expect(getEnrollmentRejectedReasonText(reason)).toMatch(expected);
  });
});
