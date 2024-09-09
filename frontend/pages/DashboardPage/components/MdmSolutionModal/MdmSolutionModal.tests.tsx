import React from "react";

import { render, screen } from "@testing-library/react";

import { IMdmSolution } from "interfaces/mdm";

import MdmSolutionModal from "./MdmSolutionModal";

describe("MdmSolutionModal table", () => {
  it("Renders table data normally", () => {
    const [server_url, hosts_count] = ["a.b.c", 123];
    const mdmSolutions: IMdmSolution[] = [
      {
        id: 1,
        name: "test solution",
        server_url,
        hosts_count,
      },
    ];
    render(
      <MdmSolutionModal mdmSolutions={mdmSolutions} onCancel={() => null} />
    );

    expect(screen.getByText(server_url)).toBeInTheDocument();
    expect(screen.getByText(hosts_count)).toBeInTheDocument();
  });
});
