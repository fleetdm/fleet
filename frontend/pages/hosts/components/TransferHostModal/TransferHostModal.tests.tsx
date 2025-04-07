import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import TransferHostModal from "./TransferHostModal";

describe("TransferHostModal", () => {
  it("renders the correct message when more than one host is being transfered", () => {
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
    ).toBeVisible();
  });

  it("render the correct message when one host is being transfered", () => {
    render(
      <TransferHostModal
        multipleHosts={false}
        teams={[]}
        onSubmit={noop}
        onCancel={noop}
        isUpdating={false}
        isGlobalAdmin={false}
      />
    );

    expect(
      screen.getByText(
        "The host's disk encryption key is deleted if it's transferred to a team with disk encryption turned off."
      )
    ).toBeVisible();
  });
});
