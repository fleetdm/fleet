import React from "react";
import { screen, waitFor, within } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { defaultConfigProfileStatusHandler } from "test/handlers/config-profiles";
import { IMdmConfig } from "interfaces/config";
import { platformToMDMLabel, ProfilePlatform } from "interfaces/mdm";

import ConfigProfileStatusModal from "./ConfigProfileStatusModal";

describe("ConfigProfileStatusModal", () => {
  const render = (ui: React.ReactElement, mdmOverride?: Partial<IMdmConfig>) =>
    createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          config: {
            mdm: {
              enabled_and_configured: true,
              ...mdmOverride,
            },
          },
        },
      },
    })(ui);

  it("renders the correct number of hosts for each status", async () => {
    mockServer.use(defaultConfigProfileStatusHandler);
    render(
      <ConfigProfileStatusModal
        name="Test profile"
        uuid="123-abc"
        teamId={0}
        platform="darwin"
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
        platform="darwin"
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

  it.each([
    "darwin",
    "ios",
    "ipados",
    "windows",
    "android",
  ] as ProfilePlatform[])(
    "shows MDM turned off for %s with correct label and links",
    async (platform) => {
      render(
        <ConfigProfileStatusModal
          name="Test profile"
          uuid="123-abc"
          teamId={0}
          platform={platform}
          onClickResend={noop}
          onExit={noop}
        />,
        { enabled_and_configured: false }
      );

      const mdmLabel = platformToMDMLabel(platform);

      await waitFor(() =>
        screen.getByText(`${mdmLabel} mdm isn't turned on`, { exact: false })
      );

      const learnMoreLink = await screen.findByRole("link", {
        name: "Learn more",
      });
      expect(learnMoreLink).toHaveAttribute(
        "href",
        expect.stringMatching(new RegExp(mdmLabel, "i"))
      );
    }
  );
});
