import React from "react";
import { render, screen } from "@testing-library/react";

import createMockSoftware from "__mocks__/softwareMock";

import Vulnerabilities from "./Vulnerabilities";

describe("Vulnerabilities", () => {
  const [mockSoftwareWithVuln, mockSoftwareNoVulns] = [
    createMockSoftware({
      vulnerabilities: [
        {
          cve: "CVE_333",
          details_link: "https://its.really.bad",
          cvss_score: 9.5,
          epss_probability: 1,
          cisa_known_exploit: false,
          cve_published: "2023-02-14T20:15:00Z",
        },
      ],
    }),
    createMockSoftware(),
  ];

  it("renders the empty state when no vulnerabilities are provided", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier
        software={mockSoftwareNoVulns}
      />
    );

    // Empty state
    expect(
      screen.getByText("No vulnerabilities detected for this software item.")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see vulnerabilities?")
    ).toBeInTheDocument();
    expect(screen.getByText("File an issue on GitHub")).toBeInTheDocument();
  });

  it("correctly renders a table when 1 vulnerability is provided, Premium tier", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier
        software={mockSoftwareWithVuln}
      />
    );

    // Rendered table
    expect(screen.getByText("Vulnerability")).toBeInTheDocument();
    expect(screen.getByText("Probability of exploit")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(screen.getByText("Known exploit")).toBeInTheDocument();
    expect(screen.getByText("Published")).toBeInTheDocument();
    expect(screen.getByText("CVE_333")).toBeInTheDocument();
    expect(screen.getByText("100%")).toBeInTheDocument();
    expect(screen.getByText("Critical", { exact: false })).toBeInTheDocument();
    expect(screen.getByText("ago", { exact: false })).toBeInTheDocument();
  });

  it("Only renders the 'Vulnerability' column when 1 vulnerability is provided on Free tier", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier={false}
        software={mockSoftwareWithVuln}
      />
    );

    // Rendered table
    expect(screen.getByText("Vulnerability")).toBeInTheDocument();

    // No premium-only columns
    expect(screen.queryByText("Probability of exploit")).toBeNull();
    expect(screen.queryByText("Severity")).toBeNull();
    expect(screen.queryByText("Known exploit")).toBeNull();
    expect(screen.queryByText("Published")).toBeNull();

    // Row data
    expect(screen.getByText("CVE_333")).toBeInTheDocument();
    expect(screen.queryByText("100%")).toBeNull();
    expect(screen.queryByText("Critical", { exact: false })).toBeNull();
    expect(screen.queryByText("ago", { exact: false })).toBeNull();
  });

  // Test for premium icons on column headers in Sandbox mode
  it("Renders 4 'Premium feature' tooltips when in premium tier Sandbox mode", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier
        isSandboxMode
        software={mockSoftwareWithVuln}
      />
    );

    expect(
      screen.getAllByText("This is a Fleet Premium feature.", { exact: false })
    ).toHaveLength(4);
  });
});
