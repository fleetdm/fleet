import { renderHook } from "@testing-library/react";
import React from "react";

import createMockUser from "__mocks__/userMock";
import { AppContext, IAppContext, initialState } from "context/app";
import { TableContext } from "context/table";
import { createMockRouter } from "test/test-utils";

import { useTeamIdParam } from "./useTeamIdParam";

describe("useTeamIdParam", () => {
  it("recognizes a global Observer+ in Unassigned", () => {
    const appContext: IAppContext = {
      ...initialState,
      currentUser: createMockUser({ global_role: "observer_plus" }),
      availableTeams: [{ id: 0, name: "Unassigned" }],
      isPremiumTier: true,
      isFreeTier: false,
    };

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <AppContext.Provider value={appContext}>
        <TableContext.Provider
          value={{ resetSelectedRows: false, setResetSelectedRows: jest.fn() }}
        >
          {children}
        </TableContext.Provider>
      </AppContext.Provider>
    );

    const { result } = renderHook(
      () =>
        useTeamIdParam({
          location: {
            pathname: "/policies/1",
            search: "?fleet_id=0",
            query: { fleet_id: "0" },
          },
          router: createMockRouter(),
          includeAllTeams: true,
          includeNoTeam: true,
        }),
      { wrapper }
    );

    expect(result.current.currentTeamId).toBe(0);
    expect(result.current.isObserverPlus).toBe(true);
  });
});
