import React from "react";
import { render, screen } from "@testing-library/react";

import Vulnerabilities from "./Vulnerabilities";

describe("Vulnerabilities", () => {
  const mockSoftwareWithVuln = {
    id: 1,
    name: "testSW",
    version: "1.0",
    source: "apps",
    generated_cpe: "a:b:c",
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
  };

  const mockSoftwareNoVulns = {
    id: 1,
    name: "testSW",
    version: "1.0",
    source: "apps",
    generated_cpe: "a:b:c",
    vulnerabilities: [],
  };

  it("renders the empty state when no vulnerabilities are provided", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier
        software={mockSoftwareNoVulns}
      />
    );

    // Header
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();

    // Empty state
    expect(
      screen.getByText("No vulnerabilities detected for this software item.")
    ).toBeInTheDocument();
    expect(
      screen.getByText("Expecting to see vulnerabilities?")
    ).toBeInTheDocument();
    expect(screen.getByText("File an issue on GitHub")).toBeInTheDocument();

    // No rendered table
    expect(screen.queryByText("Vulnerability")).toBeNull();
    expect(screen.queryByText("Probability of exploit")).toBeNull();
    expect(screen.queryByText("Severity")).toBeNull();
    expect(screen.queryByText("Known exploit")).toBeNull();
    expect(screen.queryByText("Published")).toBeNull();
  });

  it("correctly renders a table when 1 vulnerability is provided, Premium tier", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier
        software={mockSoftwareWithVuln}
      />
    );

    // Header
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();

    // No empty state
    expect(
      screen.queryByText("No vulnerabilities detected for this software item.")
    ).toBeNull();
    expect(screen.queryByText("Expecting to see vulnerabilities?")).toBeNull();
    expect(screen.queryByText("File an issue on GitHub")).toBeNull();

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
  it("correctly renders a table when 1 vulnerability is provided, Free tier", () => {
    render(
      <Vulnerabilities
        isLoading={false}
        isPremiumTier={false}
        software={mockSoftwareWithVuln}
      />
    );

    // Header
    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();

    // No empty state
    expect(
      screen.queryByText("No vulnerabilities detected for this software item.")
    ).toBeNull();
    expect(screen.queryByText("Expecting to see vulnerabilities?")).toBeNull();
    expect(screen.queryByText("File an issue on GitHub")).toBeNull();

    // Rendered table
    expect(screen.getByText("Vulnerability")).toBeInTheDocument();
    // No premium-only columns
    expect(screen.queryByText("Probability of exploit")).toBeNull();
    expect(screen.queryByText("Severity")).toBeNull();
    expect(screen.queryByText("Known exploit")).toBeNull();
    expect(screen.queryByText("Published")).toBeNull();

    expect(screen.getByText("CVE_333")).toBeInTheDocument();
    expect(screen.queryByText("100%")).toBeNull();
    expect(screen.queryByText("Critical", { exact: false })).toBeNull();
    expect(screen.queryByText("ago", { exact: false })).toBeNull();
  });
});
