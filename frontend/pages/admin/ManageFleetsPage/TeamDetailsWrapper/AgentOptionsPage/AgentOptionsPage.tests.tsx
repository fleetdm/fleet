import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
  createMockLocation,
} from "test/test-utils";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import createMockUser from "__mocks__/userMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import osqueryOptionsAPI from "services/entities/osquery_options";
import { EMPTY_AGENT_OPTIONS } from "utilities/constants";

import AgentOptionsPage from "./AgentOptionsPage";

// react-ace doesn't render an editable field under jsdom, so stand in a plain
// textarea that calls the same onChange.
jest.mock("components/YamlAce", () => ({
  __esModule: true,
  default: ({
    value,
    onChange,
    error,
  }: {
    value?: string;
    onChange: (value: string) => void;
    error?: string;
  }) => (
    <>
      <textarea
        aria-label="YAML"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {error && <span>{error}</span>}
    </>
  ),
}));

const renderAgentOptionsPage = () => {
  mockServer.use(
    createGetConfigHandler(),
    http.get(baseUrl("/fleets/:id"), () =>
      HttpResponse.json({
        team: {
          id: 1,
          name: "Team 1",
          agent_options: { config: { options: { pack_delimiter: "/" } } },
        },
      })
    )
  );

  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        isOnGlobalTeam: true,
        currentUser: createMockUser({ global_role: "admin" }),
        availableTeams: [createMockTeamSummary({ id: 1, name: "Team 1" })],
        setCurrentTeam: jest.fn(),
      },
    },
  });

  return render(
    <AgentOptionsPage
      location={createMockLocation({
        pathname: "/settings/teams/agent-options",
        search: "?fleet_id=1",
        query: { fleet_id: "1" },
      })}
      router={createMockRouter()}
    />
  );
};

describe("AgentOptionsPage", () => {
  // Regression (#52677): submitting invalid YAML threw out of yaml.load before
  // the request promise, leaving the Save button spinning forever.
  it("blocks submit while the YAML has a syntax error", async () => {
    const updateSpy = jest
      .spyOn(osqueryOptionsAPI, "updateTeam")
      .mockResolvedValue({} as never);
    const { user } = renderAgentOptionsPage();

    const editor = await screen.findByLabelText("YAML");
    await user.clear(editor);
    await user.type(editor, "config: 1\n  bad: 2");

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();

    await user.click(saveButton);
    expect(updateSpy).not.toHaveBeenCalled();
    expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
  });

  it("submits once the YAML is valid", async () => {
    const updateSpy = jest
      .spyOn(osqueryOptionsAPI, "updateTeam")
      .mockResolvedValue({} as never);
    const { user } = renderAgentOptionsPage();

    const editor = await screen.findByLabelText("YAML");
    await user.clear(editor);
    await user.type(editor, "config: null");

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(updateSpy).toHaveBeenCalled();
  });

  it("shows the syntax error on the editor label", async () => {
    const { user } = renderAgentOptionsPage();

    const editor = await screen.findByLabelText("YAML");
    await user.clear(editor);
    await user.type(editor, "config: 1\n  bad: 2");

    expect(await screen.findByText(/Syntax Error:/)).toBeInTheDocument();
  });

  it("re-enables Save once the syntax error is fixed", async () => {
    const updateSpy = jest
      .spyOn(osqueryOptionsAPI, "updateTeam")
      .mockResolvedValue({} as never);
    const { user } = renderAgentOptionsPage();

    const editor = await screen.findByLabelText("YAML");
    await user.clear(editor);
    await user.type(editor, "config: 1\n  bad: 2");
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    await user.clear(editor);
    await user.type(editor, "config: null");

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeEnabled();
    await user.click(saveButton);
    expect(updateSpy).toHaveBeenCalled();
  });

  it("submits empty agent options as an empty config", async () => {
    const updateSpy = jest
      .spyOn(osqueryOptionsAPI, "updateTeam")
      .mockResolvedValue({} as never);
    const { user } = renderAgentOptionsPage();

    const editor = await screen.findByLabelText("YAML");
    await user.clear(editor);

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(updateSpy).toHaveBeenCalledWith(1, EMPTY_AGENT_OPTIONS);
  });
});
