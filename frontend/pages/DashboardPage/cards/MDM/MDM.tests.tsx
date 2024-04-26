import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import { createMockMdmSummaryMdmSolution } from "__mocks__/mdmMock";

import MDM from "./MDM";

describe("MDM Card", () => {
  it("rolls up the data by mdm solution name and render the correct number of MDM solutions", () => {
    render(
      <MDM
        onClickMdmSolution={noop}
        error={null}
        isFetching={false}
        mdmStatusData={[]}
        mdmSolutions={[
          createMockMdmSummaryMdmSolution(),
          createMockMdmSummaryMdmSolution({ id: 2 }),
          createMockMdmSummaryMdmSolution({ name: "Test Solution", id: 3 }),
          createMockMdmSummaryMdmSolution({ name: "Test Solution", id: 4 }),
          createMockMdmSummaryMdmSolution({ name: "Test Solution 2", id: 5 }),
          // "" should render a row of "Unknown"
          createMockMdmSummaryMdmSolution({ name: "", id: 8 }),
          createMockMdmSummaryMdmSolution({ name: "", id: 9 }),
        ]}
      />
    );

    expect(screen.getAllByText("MDM Solution").length).toBe(1);
    expect(screen.getAllByText("Test Solution").length).toBe(1);
    expect(screen.getAllByText("Test Solution 2").length).toBe(1);
    expect(screen.getAllByText("Unknown").length).toBe(1);
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
