import { screen } from "@testing-library/react";
import React, { useContext, useEffect } from "react";

import createMockUser from "__mocks__/userMock";
import { renderWithSetup } from "test/test-utils";

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
    hasInvalidABMToken,
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
            hasInvalidABMToken: true,
            invalidAbmTokenOrgNames: [
              "Acme Inc.",
              "Fleet Device Management Inc.",
            ],
          })
        }
      >
        Set invalid tokens
      </button>
      <div data-testid="has-invalid">{String(hasInvalidABMToken)}</div>
      <div data-testid="org-names">{invalidAbmTokenOrgNames.join(", ")}</div>
    </div>
  );
};

describe("AppProvider - setABMExpiry", () => {
  it("defaults hasInvalidABMToken to false and invalidAbmTokenOrgNames to an empty list", () => {
    renderWithSetup(
      <AppProvider>
        <AbmExpiryConsumer />
      </AppProvider>
    );

    expect(screen.getByTestId("has-invalid")).toHaveTextContent("false");
    expect(screen.getByTestId("org-names")).toHaveTextContent("");
  });

  it("updates hasInvalidABMToken and invalidAbmTokenOrgNames when setABMExpiry is called", async () => {
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

const EJBCAExpiryConsumer = ({ expiry }: { expiry: string }) => {
  const {
    setEJBCAExpiry,
    isEJBCACertExpired,
    willEJBCACertExpire,
  } = useContext(AppContext);
  useEffect(() => {
    setEJBCAExpiry(expiry);
  }, [expiry, setEJBCAExpiry]);
  return (
    <div data-testid="ejbca">
      {String(isEJBCACertExpired)} {String(willEJBCACertExpire)}
    </div>
  );
};

describe("AppProvider - setEJBCAExpiry", () => {
  const daysFromNow = (days: number) =>
    new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();

  it.each([
    ["no certificate authority", "", "false false"],
    ["a certificate that expired", daysFromNow(-1), "true false"],
    [
      "a certificate that expires within 30 days",
      daysFromNow(10),
      "false true",
    ],
    ["a certificate that expires later", daysFromNow(90), "false false"],
  ])("flags %s", (_, expiry, want) => {
    renderWithSetup(
      <AppProvider>
        <EJBCAExpiryConsumer expiry={expiry} />
      </AppProvider>
    );
    expect(screen.getByTestId("ejbca")).toHaveTextContent(want);
  });
});
