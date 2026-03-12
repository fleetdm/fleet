import React from "react";

import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import Baselines from "./Baselines";

const mockOnMutation = jest.fn();
const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  go: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
} as any;

const baselinesHandler = http.get(baseUrl("/mdm/baselines"), () => {
  return HttpResponse.json({
    baselines: [
      {
        id: "windows-security-baseline",
        name: "Windows Security Baseline",
        version: "1.0.0",
        platform: "windows",
        description: "Corporate Windows security settings.",
        categories: [
          {
            name: "Windows Firewall",
            profiles: ["profiles/firewall.xml"],
            policies: ["policies/verify-firewall.yaml"],
            scripts: [],
          },
          {
            name: "Windows Defender",
            profiles: ["profiles/defender.xml"],
            policies: ["policies/verify-defender.yaml"],
            scripts: [],
          },
        ],
      },
    ],
  });
});

const render = createCustomRenderer({
  withBackendMock: true,
});

describe("Baselines", () => {
  beforeEach(() => {
    mockServer.use(baselinesHandler);
  });

  it("renders baseline cards when data loads", async () => {
    render(
      <Baselines
        currentTeamId={1}
        router={mockRouter}
        onMutation={mockOnMutation}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText("Windows Security Baseline")
      ).toBeInTheDocument();
    });

    expect(screen.getByText("v1.0.0")).toBeInTheDocument();
    expect(
      screen.getByText("Corporate Windows security settings.")
    ).toBeInTheDocument();
    expect(screen.getByText("2 categories")).toBeInTheDocument();
    expect(screen.getByText("2 profiles")).toBeInTheDocument();
  });

  it("renders apply button for team-scoped view", async () => {
    render(
      <Baselines
        currentTeamId={1}
        router={mockRouter}
        onMutation={mockOnMutation}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Apply to team" })
      ).toBeInTheDocument();
    });

    const applyButton = screen.getByRole("button", { name: "Apply to team" });
    expect(applyButton).not.toBeDisabled();
  });

  it("disables apply button when no team selected", async () => {
    render(
      <Baselines
        currentTeamId={0}
        router={mockRouter}
        onMutation={mockOnMutation}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "Apply to team" })
      ).toBeInTheDocument();
    });

    const applyButton = screen.getByRole("button", { name: "Apply to team" });
    expect(applyButton).toBeDisabled();
    expect(
      screen.getByText("Select a team to apply or remove this baseline.")
    ).toBeInTheDocument();
  });

  it("renders section header and description", async () => {
    render(
      <Baselines
        currentTeamId={1}
        router={mockRouter}
        onMutation={mockOnMutation}
      />
    );

    await waitFor(() => {
      expect(screen.getByText("Security baselines")).toBeInTheDocument();
    });
  });
});
