import React from "react";
import { screen, within } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { defaultConfigProfileStatusHandler } from "test/handlers/config-profiles";

import ConfigProfileStatusModal from "./ConfigProfileStatusModal";

describe("ConfigProfileStatusModal", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the correct number of hosts for each status", async () => {
    mockServer.use(defaultConfigProfileStatusHandler);
    render(
      <ConfigProfileStatusModal
        name="Test profile"
        uuid="123-abc"
        teamId={0}
        onClickResend={noop}
        onExit={noop}
      />
    );

    await screen.findByText("Verified");

    // get all rows in the table and skip header row
    const rows = screen.getAllByRole("row").slice(1);

    const verifiedRow = within(rows[0]).getAllByRole("cell");
    expect(verifiedRow[0]).toHaveTextContent("Verified");
    expect(verifiedRow[1]).toHaveTextContent("---");

    const verifiyingRow = within(rows[1]).getAllByRole("cell");
    expect(verifiyingRow[0]).toHaveTextContent("Verifying");
    expect(verifiyingRow[1]).toHaveTextContent("1");

    const pendingRow = within(rows[2]).getAllByRole("cell");
    expect(pendingRow[0]).toHaveTextContent("Pending");
    expect(pendingRow[1]).toHaveTextContent("2");

    const failedRow = within(rows[3]).getAllByRole("cell");
    expect(failedRow[0]).toHaveTextContent("Failed");
    expect(failedRow[1]).toHaveTextContent("3");
  });

  it("shows the resend button for a failed row on hover", async () => {
    mockServer.use(defaultConfigProfileStatusHandler);
    const { user } = render(
      <ConfigProfileStatusModal
        name="Test profile"
        uuid="123-abc"
        teamId={0}
        onClickResend={noop}
        onExit={noop}
      />
    );

    await screen.findByText("Verified");

    const failedRow = screen.getByText("Failed").closest("tr");
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    user.hover(failedRow!);

    const resendButton = screen.getByRole("button", { name: "Resend" });
    expect(resendButton).toBeVisible();
  });
});
