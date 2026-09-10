import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";
import { PolicyResponse } from "utilities/constants";
import { HostStatusFilter } from "interfaces/host";

import HostsFilterBlock from "./HostsFilterBlock";

const render = createCustomRenderer({
  context: {
    app: {
      currentUser: createMockUser({ global_role: "admin" }),
      isOnGlobalTeam: true,
    },
  },
});

const noop = () => undefined;

const renderWithStatus = (
  status?: HostStatusFilter,
  handleClearFilter = jest.fn()
) =>
  render(
    <HostsFilterBlock
      params={{
        munkiIssueDetails: null,
        policyResponse: PolicyResponse.PASSING,
        softwareDetails: null,
        mdmSolutionDetails: null,
        scriptBatchRanAt: null,
        scriptBatchScriptName: null,
        depProfileError: "",
        status,
      }}
      handleClearRouteParam={noop}
      handleClearFilter={handleClearFilter}
      onChangePoliciesFilter={noop}
      onChangeOsSettingsFilter={noop}
      onChangeDiskEncryptionStatusFilter={noop}
      onChangeBootstrapPackageStatusFilter={noop}
      onChangeMacSettingsFilter={noop}
      onChangeSoftwareInstallStatusFilter={noop}
      onChangeConfigProfileStatusFilter={noop}
      onChangeScriptBatchStatusFilter={noop}
      onClickEditLabel={noop}
      onClickDeleteLabel={noop}
    />
  );

describe("HostsFilterBlock", () => {
  describe("enrolled status filter", () => {
    it("shows a clearable pill so the drill-down view reads as filtered", async () => {
      const handleClearFilter = jest.fn();
      const { user } = renderWithStatus("enrolled", handleClearFilter);

      expect(screen.getByText("Status: Enrolled")).toBeInTheDocument();

      await user.click(
        screen.getByRole("button", { name: "Remove Status: Enrolled filter" })
      );
      expect(handleClearFilter).toHaveBeenCalledWith(["status"]);
    });

    it("does not render a pill for statuses picked from the dropdown", () => {
      renderWithStatus("online");

      expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });
  });
});
