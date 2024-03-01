import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import { createMockMdmSolution } from "__mocks__/mdmMock";

import MDM from "./MDM";

describe("MDM Card", () => {
  it("rolls up the data by mdm solution name and render the correct number of MDM solutions", () => {
    const { debug } = render(
      <MDM
        onClickMdmSolution={noop}
        error={null}
        isFetching={false}
        mdmStatusData={[]}
        mdmSolutions={[
          createMockMdmSolution(),
          createMockMdmSolution({ id: 2 }),
          createMockMdmSolution({ name: "Test Solution", id: 3 }),
          createMockMdmSolution({ name: "Test Solution", id: 4 }),
          createMockMdmSolution({ name: "Test Solution 2", id: 5 }),
        ]}
      />
    );

    debug();

    expect(screen.getAllByText("MDM Solution").length).toBe(1);
    expect(screen.getAllByText("Test Solution").length).toBe(1);
    expect(screen.getAllByText("Test Solution 2").length).toBe(1);
  });

  it("render the correct number of Enrollment status", async () => {
    const { user } = renderWithSetup(
      <MDM
        onClickMdmSolution={noop}
        error={null}
        isFetching={false}
        mdmStatusData={[
          { status: "On (automatic)", hosts: 10 },
          { status: "On (manual)", hosts: 5 },
          { status: "Off", hosts: 1 },
          { status: "Pending", hosts: 3 },
        ]}
        mdmSolutions={[]}
      />
    );

    await user.click(screen.getByRole("tab", { name: "Status" }));

    expect(
      screen.getByRole("row", {
        name: /On \(automatic\)(.*?)10 host/i,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", {
        name: /On \(manual\)(.*?)5 host/i,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", {
        name: /Off(.*?)1 host/i,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", {
        name: /Pending(.*?)3 host/i,
      })
    ).toBeInTheDocument();
  });
});
