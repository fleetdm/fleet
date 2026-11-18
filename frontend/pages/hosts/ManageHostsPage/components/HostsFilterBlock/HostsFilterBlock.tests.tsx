import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockLabel } from "__mocks__/labelsMock";
import { createMockOSVersion } from "__mocks__/operatingSystemsMock";
import { createMockSoftware } from "__mocks__/softwareMock";
import { createCustomRenderer } from "test/test-utils";
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

  describe("operating system pill", () => {
    const osVersions = [
      createMockOSVersion({
        name: "Windows 11 Pro 24H2 10.0.26100.1742",
        name_only: "Windows 11 Pro 24H2",
        version: "10.0.26100.1742",
        platform: "windows",
      }),
    ];

    it.each([
      // Matches the OS version entry.
      [
        "Windows 11 Pro 24H2",
        "10.0.26100.1742",
        "Windows 11 Pro 24H2 10.0.26100.1742",
      ],
      // No entry: a distro the OS inventory groups under "Arch Linux", and a
      // saved link with the older Windows name.
      ["Omarchy", "4.0.0", "Omarchy 4.0.0"],
      [
        "Microsoft Windows 11 Pro 24H2",
        "10.0.26100.1742",
        "Microsoft Windows 11 Pro 24H2 10.0.26100.1742",
      ],
    ])("labels the %p %p filter as %p", (osName, osVersion, expected) => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{ ...baseParams, osName, osVersion, osVersions }}
        />
      );

      expect(screen.getByText(expected)).toBeInTheDocument();
    });
  });

  describe("label combined with an Apple Business filter", () => {
    const selectedLabel = createMockLabel({ display_text: "Engineering" });

    it("shows both the label and the AB issues pills", () => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{ ...baseParams, depProfileError: "true" }}
          selectedLabel={selectedLabel}
        />
      );

      expect(screen.getByText("Engineering")).toBeInTheDocument();
      expect(
        screen.getByText("Apple Business (AB) issues")
      ).toBeInTheDocument();
    });

    it("shows only the label pill when the AB issues filter is false", () => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{ ...baseParams, depProfileError: "false" }}
          selectedLabel={selectedLabel}
        />
      );

      expect(screen.getByText("Engineering")).toBeInTheDocument();
      expect(
        screen.queryByText("Apple Business (AB) issues")
      ).not.toBeInTheDocument();
    });

    it("shows both the label and the AB profile assignment pills", () => {
      render(
        <HostsFilterBlock
          {...baseProps}
          params={{ ...baseParams, depAssignProfileResponse: "FAILED" }}
          selectedLabel={selectedLabel}
        />
      );

      expect(screen.getByText("Engineering")).toBeInTheDocument();
      expect(
        screen.getByText("Apple Business (AB) issue: Failed")
      ).toBeInTheDocument();
    });
  });
});
