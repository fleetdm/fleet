import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";

import {
  IMunkiIssuesAggregate,
  IMunkiVersionsAggregate,
} from "interfaces/macadmins";
import Munki from "./Munki";

describe("Munki card", () => {
  it("renders data normally when present", async () => {
    const [name1, name2] = ["Munki issue 1", "Munki issue 2"];
    const [
      errorMacAdmins,
      isMacAdminsFetching,
      munkiIssuesData,
      munkiVersionsData,
      selectedTeamId,
    ] = [
      null,
      false,
      [
        {
          id: 476,
          name: name1,
          type: "warning",
          hosts_count: 2345,
        },
        {
          id: 555,
          name: name2,
          type: "error",
          hosts_count: 5432,
        },
      ] as IMunkiIssuesAggregate[],
      [
        {
          version: "1.2.3",
          hosts_count: 37,
        },
      ] as IMunkiVersionsAggregate[],
      undefined,
    ];

    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          currentUser: createMockUser(),
        },
      },
    });

    const { user } = render(
      <Munki
        errorMacAdmins={errorMacAdmins}
        isMacAdminsFetching={isMacAdminsFetching}
        munkiIssuesData={munkiIssuesData}
        munkiVersionsData={munkiVersionsData}
        selectedTeamId={selectedTeamId}
      />
    );

    // Issues tab
    expect(screen.getByText("Issue")).toBeInTheDocument();
    expect(screen.getByText("Type")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();

    expect(screen.getAllByText(name1)).toHaveLength(2);
    expect(screen.getByText("Warning")).toBeInTheDocument();
    expect(screen.getByText("2345")).toBeInTheDocument();

    expect(screen.getAllByText(name2)).toHaveLength(2);
    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getByText("5432")).toBeInTheDocument();

    // Versions tab

    await user.click(screen.getByText("Versions"));

    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Hosts")).toBeInTheDocument();
    expect(screen.getByText("1.2.3")).toBeInTheDocument();
    expect(screen.getByText("37")).toBeInTheDocument();
  });
});
