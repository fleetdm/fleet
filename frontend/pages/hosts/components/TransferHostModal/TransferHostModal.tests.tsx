import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import TransferHostModal from "./TransferHostModal";

describe("TransferHostModal", () => {
  it("does not render the disk encryption message", () => {
    render(
      <TransferHostModal
        multipleHosts
        teams={[]}
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
        isGlobalAdmin={false}
      />
    );

    expect(
      screen.getByText(
        "The hosts' disk encryption keys are deleted if they're transferred to a team with disk encryption turned off."
      )
    ).not.toBeVisible();
  });
});
