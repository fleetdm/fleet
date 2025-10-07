import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";

import AddInstallSoftware from "./AddInstallSoftware";

describe("AddInstallSoftware", () => {
  it("should render the expected message if there are no software titles to select from", () => {
    render(
      <AddInstallSoftware
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={null}
        onAddSoftware={noop}
        hasManualAgentInstall={false}
        platform="macos"
      />
    );

    expect(screen.getByText(/you can add software on the/i)).toBeVisible();
  });

  it("should render the correct messaging when there are software titles but none have been selected to install at setup", () => {
    render(
      <AddInstallSoftware
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle(), createMockSoftwareTitle()]}
        onAddSoftware={noop}
        hasManualAgentInstall={false}
        platform="macos"
      />
    );

    expect(screen.getByText(/No software selected/)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Add software" })).toBeNull();
  });

  it("should render the correct messaging when there are software titles that have been selected to install at setup", () => {
    render(
      <AddInstallSoftware
        savedRequireAllSoftwareMacOS={false}
        currentTeamId={1}
        softwareTitles={[
          createMockSoftwareTitle({
            software_package: createMockSoftwarePackage({
              install_during_setup: true,
            }),
          }),
          createMockSoftwareTitle(
            createMockSoftwareTitle({
              software_package: createMockSoftwarePackage({
                install_during_setup: true,
              }),
            })
          ),
          createMockSoftwareTitle(),
        ]}
        onAddSoftware={noop}
        hasManualAgentInstall={false}
        platform="macos"
      />
    );

    expect(
      screen.getByText(
        (_, element) =>
          element?.textContent ===
          "2 software items will be installed during setup."
      )
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Select software" })
    ).toBeVisible();
  });
});
