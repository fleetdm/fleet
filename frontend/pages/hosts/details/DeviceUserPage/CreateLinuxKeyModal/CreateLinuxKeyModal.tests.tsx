import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import CreateLinuxKeyModal from "./CreateLinuxKeyModal";

const renderModal = (props: {
  isEscrowInFlight: boolean;
  retryAfterSeconds?: number;
}) =>
  render(
    <CreateLinuxKeyModal
      isTriggeringCreateLinuxKey={false}
      onExit={noop}
      {...props}
    />
  );

describe("CreateLinuxKeyModal", () => {
  it("tells the end user to expect a pop-up when the request was queued", () => {
    renderModal({ isEscrowInFlight: false });

    expect(screen.getByText(/Wait 30 seconds/i)).toBeVisible();
    expect(screen.queryByText(/already asking/i)).toBeNull();
  });

  it("covers each state the prompt can be in when the escrow is already in flight", () => {
    renderModal({ isEscrowInFlight: true });

    expect(screen.getByText(/already asking/i)).toBeVisible();
    // the server cannot tell which state the prompt is in, so all three are covered
    expect(screen.getByText(/pop-up is open/i)).toBeVisible();
    expect(screen.getByText(/already entered your passphrase/i)).toBeVisible();
    expect(
      screen.getByText(/before you entered your passphrase/i)
    ).toBeVisible();
    expect(screen.getByText(/TPM-backed/i)).toBeVisible();
    expect(screen.queryByText(/Wait 30 seconds/i)).toBeNull();
    // no Retry-After from the server, so the wait stays generic
    expect(screen.getByText(/wait a few minutes/i)).toBeVisible();
  });

  it.each([
    // rounded up to whole minutes, never down, with the singular at one minute
    { retryAfterSeconds: 90, expected: /wait 2 minutes/i },
    { retryAfterSeconds: 20, expected: /wait 1 minute,/i },
  ])(
    "tells the end user to wait $retryAfterSeconds seconds as whole minutes",
    ({ retryAfterSeconds, expected }) => {
      renderModal({ isEscrowInFlight: true, retryAfterSeconds });

      expect(screen.getByText(expected)).toBeVisible();
    }
  );
});
