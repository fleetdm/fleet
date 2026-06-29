import React from "react";
import { render, screen } from "@testing-library/react";

import { IPolicy } from "interfaces/policy";
import ENDPOINTS from "utilities/endpoints";

import PolicyAutomationsList from "./PolicyAutomationsList";

// Stub SoftwareIcon to avoid asset resolution in tests; surface the url and
// name props so we can assert the raw software name (used for fallback icon
// matching) and the custom icon path are forwarded.
jest.mock("pages/SoftwarePage/components/icons/SoftwareIcon", () => {
  return ({ name, url }: { name?: string; url?: string | null }) => (
    <span
      data-testid="software-icon"
      data-name={name ?? ""}
      data-url={url ?? ""}
    />
  );
});

const createMockPolicy = (overrides?: Partial<IPolicy>): IPolicy => ({
  id: 1,
  name: "Test policy",
  query: "SELECT 1;",
  description: "",
  author_id: 1,
  author_name: "Admin",
  author_email: "admin@example.com",
  resolution: "",
  platform: "darwin",
  team_id: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  ...overrides,
});

describe("PolicyAutomationsList", () => {
  describe("automations list", () => {
    it("shows empty state when no automations are configured", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy()}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("No automations")).toBeInTheDocument();
    });

    it("shows software automation row", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: { name: "Zoom", software_title_id: 42 },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("Zoom")).toBeInTheDocument();
      expect(screen.queryByText("No automations")).not.toBeInTheDocument();
    });

    it("prefers install_software.display_name over name", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: {
              name: "Zoom.pkg",
              display_name: "Zoom Workplace",
              software_title_id: 42,
            },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("Zoom Workplace")).toBeInTheDocument();
      expect(screen.queryByText("Zoom.pkg")).not.toBeInTheDocument();
    });

    it("normalizes well-known software names via the display helper", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: {
              name: "Microsoft.CompanyPortal",
              software_title_id: 42,
            },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("Company Portal")).toBeInTheDocument();
      expect(
        screen.queryByText("Microsoft.CompanyPortal")
      ).not.toBeInTheDocument();
    });

    it("forwards the custom software icon_url to SoftwareIcon", () => {
      const iconUrl = ENDPOINTS.SOFTWARE_ICON(42);
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: {
              name: "Zoom",
              software_title_id: 42,
              icon_url: iconUrl,
            },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByTestId("software-icon")).toHaveAttribute(
        "data-url",
        iconUrl
      );
    });

    it("renders SoftwareIcon with no url when there is no custom icon", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: { name: "Zoom", software_title_id: 42 },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByTestId("software-icon")).toHaveAttribute(
        "data-url",
        ""
      );
    });

    it("passes the raw install_software.name to SoftwareIcon even when a display_name override is set (regression: #47123)", () => {
      // The display name is what users see, but SoftwareIcon's fallback
      // matcher needs the raw name to find FMA / well-known icons; otherwise
      // a custom display name causes it to fall through to the generic icon.
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            install_software: {
              name: "Zoom",
              display_name: "Custom Renamed App",
              software_title_id: 42,
            },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByTestId("software-icon")).toHaveAttribute(
        "data-name",
        "Zoom"
      );
      // sanity check: label still uses the display name override
      expect(screen.getByText("Custom Renamed App")).toBeInTheDocument();
    });

    it("shows script automation row", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            run_script: { id: 1, name: "fix.sh" },
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("fix.sh")).toBeInTheDocument();
    });

    it("shows calendar automation row", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({ calendar_events_enabled: true })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("Maintenance window")).toBeInTheDocument();
    });

    it("shows conditional access automation row", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({
            conditional_access_enabled: true,
          })}
          currentAutomatedPolicies={[]}
        />
      );

      expect(screen.getByText("Block single sign-on")).toBeInTheDocument();
    });

    it("shows 'Webhook' for other automation when otherAutomationType is webhook", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
          otherAutomationType="webhook"
        />
      );

      expect(screen.getByText("Webhook")).toBeInTheDocument();
      expect(screen.queryByText("Ticket")).not.toBeInTheDocument();
    });

    it("shows 'Ticket' for other automation when otherAutomationType is ticket", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
          otherAutomationType="ticket"
        />
      );

      expect(screen.getByText("Ticket")).toBeInTheDocument();
      expect(screen.queryByText("Webhook")).not.toBeInTheDocument();
    });

    it("shows 'Webhook or ticket' for other automation when otherAutomationType is not set", () => {
      render(
        <PolicyAutomationsList
          storedPolicy={createMockPolicy({ id: 1 })}
          currentAutomatedPolicies={[1]}
        />
      );

      expect(screen.getByText("Webhook or ticket")).toBeInTheDocument();
    });
  });
});
