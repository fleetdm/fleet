import React from "react";
import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import { createMockMdmSolution } from "__mocks__/mdmMock";

import MDM from "./MDM";

describe("MDM Card", () => {
  it("render the correct number of MDM solutions", () => {
    render(
      <MDM
        error={null}
        isFetching={false}
        mdmEnrollmentData={[]}
        mdmSolutions={[
          createMockMdmSolution(),
          createMockMdmSolution({ id: 2 }),
        ]}
      />
    );

    expect(screen.getAllByText("MDM Solution").length).toBe(2);
  });

  it("render the correct number of Enrollment status", async () => {
    const { user } = renderWithSetup(
      <MDM
        error={null}
        isFetching={false}
        mdmEnrollmentData={[
          { status: "Enrolled (automatic)", hosts: 10 },
          { status: "Enrolled (manual)", hosts: 5 },
          { status: "Unenrolled", hosts: 1 },
        ]}
        mdmSolutions={[]}
      />
    );

    await user.click(screen.getByRole("tab", { name: "Enrollment" }));

    expect(
      screen.getByRole("row", {
        name: /Enrolled \(automatic\)(.*?)10 host/i,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", {
        name: /Enrolled \(manual\)(.*?)5 host/i,
      })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("row", {
        name: /Unenrolled(.*?)1 host/i,
      })
    ).toBeInTheDocument();
  });
});
