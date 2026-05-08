import React from "react";
import { screen, waitFor } from "@testing-library/react";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import { http, HttpResponse } from "msw";

import ScriptLibrary from "./ScriptLibrary";
import { ScriptsLocation } from "../../Scripts";

const mockRouter = createMockRouter();

const mockLocation: ScriptsLocation = {
  pathname: "/controls/scripts/library",
  query: {},
  search: "",
};

const emptyScriptsHandler = http.get(baseUrl("/scripts"), () =>
  HttpResponse.json({
    scripts: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const baseProps = {
  router: mockRouter,
  teamId: 1,
  location: mockLocation,
};

describe("ScriptLibrary empty state", () => {
  it("renders Upload CTA and info text for global admin", async () => {
    mockServer.use(emptyScriptsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          config: {
            server_settings: { scripts_disabled: false },
          },
        },
      },
    });

    render(<ScriptLibrary {...baseProps} />);

    await waitFor(() => {
      expect(screen.getByText("No scripts")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /upload/i })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Upload shell \(.sh\) or Python \(.py\)/i)
    ).toBeInTheDocument();
  });

  it("renders Upload CTA even when scripts are disabled (managing library is still allowed)", async () => {
    mockServer.use(emptyScriptsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalAdmin: true,
          config: {
            server_settings: { scripts_disabled: true },
          },
        },
      },
    });

    render(<ScriptLibrary {...baseProps} />);

    await waitFor(() => {
      expect(screen.getByText("No scripts")).toBeInTheDocument();
    });
    expect(
      screen.getByRole("button", { name: /upload/i })
    ).toBeInTheDocument();
  });

  it("hides Upload CTA and info text for global technician", async () => {
    mockServer.use(emptyScriptsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isGlobalTechnician: true,
          config: {
            server_settings: { scripts_disabled: false },
          },
        },
      },
    });

    render(<ScriptLibrary {...baseProps} />);

    await waitFor(() => {
      expect(screen.getByText("No scripts")).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", { name: /upload/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Upload shell \(.sh\) or Python \(.py\)/i)
    ).not.toBeInTheDocument();
  });

  it("hides Upload CTA and info text for team technician", async () => {
    mockServer.use(emptyScriptsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isTeamTechnician: true,
          config: {
            server_settings: { scripts_disabled: false },
          },
        },
      },
    });

    render(<ScriptLibrary {...baseProps} />);

    await waitFor(() => {
      expect(screen.getByText("No scripts")).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", { name: /upload/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Upload shell \(.sh\) or Python \(.py\)/i)
    ).not.toBeInTheDocument();
  });
});
