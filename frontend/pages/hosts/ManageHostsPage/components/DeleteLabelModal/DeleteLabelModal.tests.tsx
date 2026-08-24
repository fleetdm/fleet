import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import DeleteLabelModal from "./DeleteLabelModal";

const baseProps = {
  onSubmit: jest.fn(),
  onCancel: jest.fn(),
  isUpdatingLabel: false,
};

describe("DeleteLabelModal", () => {
  it("warns about configuration profiles, software targets, and reports/policies on premium tier", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
        },
      },
    });

    render(<DeleteLabelModal {...baseProps} />);

    expect(
      screen.getByText(/targeted in a configuration profile/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/used in custom software targets/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(/reports and policies that target this label/i)
    ).toBeInTheDocument();
  });

  it("does not show premium warnings on free tier", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
        },
      },
    });

    render(<DeleteLabelModal {...baseProps} />);

    expect(
      screen.queryByText(/used in custom software targets/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/targeted in a configuration profile/i)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/reports and policies that target this label/i)
    ).not.toBeInTheDocument();
  });
});
