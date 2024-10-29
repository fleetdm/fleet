import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import {
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";

import AddInstallSoftware from "./AddInstallSoftware";

describe("AddInstallSoftware", () => {
  it("should render no software message if there are no software to select from", () => {
    render(
      <AddInstallSoftware
        currentTeamId={1}
        softwareTitles={null}
        onAddSoftware={noop}
      />
    );

    expect(screen.getByText(/No software available to add/i)).toBeVisible();
    expect(screen.getByText(/upload software/i)).toBeVisible();
  });

  it("should render the correct messaging when there are software titles but none have been selected to install at setup", () => {
    render(
      <AddInstallSoftware
        currentTeamId={1}
        softwareTitles={[createMockSoftwareTitle(), createMockSoftwareTitle()]}
        onAddSoftware={noop}
      />
    );

    expect(screen.getByText(/No software added/)).toBeVisible();
    expect(screen.getByRole("button", { name: "Add software" })).toBeVisible();
  });

  it("should render the correct messaging when there are software titles that have been selected to install at setup", () => {
    render(
      <AddInstallSoftware
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
      />
    );

    expect(
      screen.getByText(/2 software will be installed during setup/)
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Show selected software" })
    ).toBeVisible();
  });
});
