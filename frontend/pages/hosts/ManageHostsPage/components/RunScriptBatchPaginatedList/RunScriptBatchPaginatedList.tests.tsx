import React from "react";

import { http, HttpResponse } from "msw";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { baseUrl, createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";

import { createMockScript } from "__mocks__/scriptMock";

import RunScriptBatchPaginatedList from "./RunScriptBatchPaginatedList";

const waitForLoadingToFinish = async (container: HTMLElement) => {
  await waitFor(() => {
    expect(container.querySelector(".loading-overlay")).not.toBeInTheDocument();
  });
};

const team1Scripts = [
  createMockScript({ team_id: 1, name: "Team script 1" }),
  createMockScript({ id: 2, team_id: 1, name: "Team script 2" }),
];

const teamScriptsHandler = http.get(baseUrl(`/scripts?team_id=1`), () => {
  // The case where a team has no scripts is handled by the parent
  return HttpResponse.json({
    scripts: team1Scripts,
  });
});

describe("RunScriptBatchPaginatedList - component", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("Lists a team's scripts", async () => {
    mockServer.use(teamScriptsHandler);
    const { container } = render(
      <RunScriptBatchPaginatedList
        onRunScript={jest.fn()}
        isUpdating={false}
        teamId={1}
        scriptCount={2}
        setScriptForDetails={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);

    const listedScripts = screen.getAllByRole("listitem");
    expect(listedScripts).toHaveLength(team1Scripts.length);
    team1Scripts.forEach((item, index) => {
      expect(listedScripts[index]).toHaveTextContent(item.name);
    });
  });
  // });

  it("Calls `onRunScript` with the appropriate script when `Run script`/`Run again` is clicked", async () => {
    mockServer.use(teamScriptsHandler);
    const onRunScript = jest.fn();
    const { container } = render(
      <RunScriptBatchPaginatedList
        onRunScript={onRunScript}
        isUpdating={false}
        teamId={1}
        scriptCount={2}
        setScriptForDetails={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);
    const listedScripts = screen.getAllByRole("listitem");
    await userEvent.click(within(listedScripts[0]).getByRole("button"));
    await waitFor(() => {
      expect(onRunScript.mock.calls.length).toEqual(1); //
    });
    // checking ids rather than full equality allows extending the components `fetchPage` to
    // modifying the incoming scripst without breaking this test
    //       const changedItems = onSubmit.mock.calls[0][0];
    const ranScript = onRunScript.mock.calls[0][0]; // the second arg is a callback
    expect(ranScript.id).toEqual(team1Scripts[0].id);
  });

  it("Sets the right script for details when clicking on the script's name", async () => {
    mockServer.use(teamScriptsHandler);
    const onSetScriptForDetails = jest.fn();
    const { container } = render(
      <RunScriptBatchPaginatedList
        onRunScript={jest.fn()}
        isUpdating={false}
        teamId={1}
        scriptCount={2}
        setScriptForDetails={onSetScriptForDetails}
      />
    );
    await waitForLoadingToFinish(container);

    const listedScripts = screen.getAllByRole("listitem");
    // click on the script's name
    await userEvent.click(
      within(listedScripts[0]).getByText(team1Scripts[0].name)
    );
    await waitFor(() => {
      expect(onSetScriptForDetails.mock.calls.length).toEqual(1); //
    });
    // checking ids rather than full equality allows extending the components `fetchPage` to
    // modifying the incoming scripst without breaking this test
    const detailsScript = onSetScriptForDetails.mock.calls[0][0]; // the second arg is a callback
    expect(detailsScript.id).toEqual(team1Scripts[0].id);
  });
});
