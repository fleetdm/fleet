import React, { useContext } from "react";

import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import createMockUser from "__mocks__/userMock";

import AppProvider, { AppContext, sortAvailableTeams } from "./app";

describe("sortAvailableTeams", () => {
  it("places Unassigned last for global team users", () => {
    const teams = [
      { id: 0, name: "Unassigned" },
      { id: 2, name: "Zebra" },
      { id: 1, name: "Alpha" },
      { id: -1, name: "All fleets" },
    ];
    const result = sortAvailableTeams(teams, createMockUser());
    expect(result.map((t) => t.name)).toEqual([
      "All fleets",
      "Alpha",
      "Zebra",
      "Unassigned",
    ]);
  });

  it("does not include All fleets or Unassigned for non-global users", () => {
    const teams = [
      { id: 0, name: "Unassigned" },
      { id: 2, name: "Zebra" },
      { id: 1, name: "Alpha" },
      { id: -1, name: "All fleets" },
    ];
    const result = sortAvailableTeams(
      teams,
      createMockUser({ global_role: null })
    );
    expect(result.map((t) => t.name)).toEqual(["Alpha", "Zebra"]);
  });

  it("sorts named teams alphabetically (case-insensitive)", () => {
    const teams = [
      { id: 3, name: "charlie" },
      { id: 1, name: "Alpha" },
      { id: 2, name: "Bravo" },
    ];
    const result = sortAvailableTeams(
      teams,
      createMockUser({ global_role: null })
    );
    expect(result.map((t) => t.name)).toEqual(["Alpha", "Bravo", "charlie"]);
  });
});

const AbmExpiryConsumer = () => {
  const {
    hasAbmTokenInvalid,
    invalidAbmTokenOrgNames,
    setABMExpiry,
  } = useContext(AppContext);

  return (
    <div>
      <button
        type="button"
        onClick={() =>
          setABMExpiry({
            earliestExpiry: "",
            needsAbmTermsRenewal: false,
            hasAbmTokenInvalid: true,
            invalidAbmTokenOrgNames: [
              "Acme Inc.",
              "Fleet Device Management Inc.",
            ],
          })
        }
      >
        Set invalid tokens
      </button>
      <div data-testid="has-invalid">{String(hasAbmTokenInvalid)}</div>
      <div data-testid="org-names">{invalidAbmTokenOrgNames.join(", ")}</div>
    </div>
  );
};

describe("AppProvider - setABMExpiry", () => {
  it("defaults hasAbmTokenInvalid to false and invalidAbmTokenOrgNames to an empty list", () => {
    renderWithSetup(
      <AppProvider>
        <AbmExpiryConsumer />
      </AppProvider>
    );

    expect(screen.getByTestId("has-invalid")).toHaveTextContent("false");
    expect(screen.getByTestId("org-names")).toHaveTextContent("");
  });

  it("updates hasAbmTokenInvalid and invalidAbmTokenOrgNames when setABMExpiry is called", async () => {
    const { user } = renderWithSetup(
      <AppProvider>
        <AbmExpiryConsumer />
      </AppProvider>
    );

    await user.click(
      screen.getByRole("button", { name: "Set invalid tokens" })
    );

    expect(screen.getByTestId("has-invalid")).toHaveTextContent("true");
    expect(screen.getByTestId("org-names")).toHaveTextContent(
      "Acme Inc., Fleet Device Management Inc."
    );
  });
});
