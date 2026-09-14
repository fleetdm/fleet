import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockSoftware } from "__mocks__/softwareMock";
import { PolicyResponse } from "utilities/constants";

import HostsFilterBlock from "./HostsFilterBlock";

const baseParams = {
  munkiIssueDetails: null,
  policyResponse: PolicyResponse.PASSING,
  softwareDetails: null,
  mdmSolutionDetails: null,
  scriptBatchRanAt: null,
  scriptBatchScriptName: null,
  depProfileError: "",
};

const baseProps = {
  handleClearRouteParam: noop,
  handleClearFilter: noop,
  onChangePoliciesFilter: noop,
  onChangeOsSettingsFilter: noop,
  onChangeDiskEncryptionStatusFilter: noop,
  onChangeBootstrapPackageStatusFilter: noop,
  onChangeMacSettingsFilter: noop,
  onChangeSoftwareInstallStatusFilter: noop,
  onChangeConfigProfileStatusFilter: noop,
  onChangeScriptBatchStatusFilter: noop,
  onClickEditLabel: noop,
  onClickDeleteLabel: noop,
};

describe("HostsFilterBlock", () => {
  const render = createCustomRenderer({
    context: { app: { isOnGlobalTeam: true } },
  });

  describe("software version pill", () => {
    it("appends the Go toolchain version for a go_binaries version", () => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{
            ...baseParams,
            softwareVersionId: 7,
            softwareDetails: createMockSoftware({
              name: "gopls",
              version: "v0.21.1",
              release: "go1.26.1",
              source: "go_binaries",
            }),
          }}
        />
      );

      expect(screen.getByText("gopls v0.21.1 (go1.26.1)")).toBeInTheDocument();
    });

    it("renders the plain version for a source that also populates release", () => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{
            ...baseParams,
            softwareVersionId: 7,
            softwareDetails: createMockSoftware({
              name: "openssl",
              version: "1.1.1k",
              release: "30.el7",
              source: "rpm_packages",
            }),
          }}
        />
      );

      expect(screen.getByText("openssl 1.1.1k")).toBeInTheDocument();
    });
  });
});
