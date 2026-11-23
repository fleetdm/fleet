import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import ExternalIdentitiesSection from "./ExternalIdentitiesSection";

const renderWithSetup = createCustomRenderer({});

describe("ExternalIdentitiesSection", () => {
  const defaultProps = {
    identities: [{ issuer: "github", subject: "repo:acme/mdm" }],
    issuers: ["github", "google"],
    onChange: noop,
    getError: () => undefined,
    onSubjectFocus: noop,
  };

  it("renders each bound identity", () => {
    renderWithSetup(<ExternalIdentitiesSection {...defaultProps} />);

    expect(screen.getByText("External identities")).toBeInTheDocument();
    expect(screen.getByText("github")).toBeInTheDocument();
    expect(screen.getByDisplayValue("repo:acme/mdm")).toBeInTheDocument();
  });

  it("adds a row defaulting to the first issuer", async () => {
    const onChange = jest.fn();
    const { user } = renderWithSetup(
      <ExternalIdentitiesSection {...defaultProps} onChange={onChange} />
    );

    await user.click(
      screen.getByRole("button", { name: /Add external identity$/ })
    );

    expect(onChange).toHaveBeenCalledWith([
      { issuer: "github", subject: "repo:acme/mdm" },
      { issuer: "github", subject: "" },
    ]);
  });

  it("removes a row", async () => {
    const onChange = jest.fn();
    const { user } = renderWithSetup(
      <ExternalIdentitiesSection {...defaultProps} onChange={onChange} />
    );

    await user.click(
      screen.getByRole("button", { name: "Remove external identity 1" })
    );

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("shows a row's error in its Subject label", () => {
    renderWithSetup(
      <ExternalIdentitiesSection
        {...defaultProps}
        getError={(name) =>
          name === "external_identities.0.subject"
            ? "Enter a subject"
            : undefined
        }
      />
    );

    expect(screen.getByText("Enter a subject")).toBeInTheDocument();
  });
});
