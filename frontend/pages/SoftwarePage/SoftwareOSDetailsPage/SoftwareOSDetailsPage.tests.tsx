// __tests__/SoftwareOSDetailsCards.test.tsx
import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockRouter } from "test/test-utils";
import {
  createMockLinuxOSVersion,
  createMockOSVersion,
} from "__mocks__/operatingSystemsMock";

import {
  SummaryCard,
  VulnerabilitiesCard,
  KernelsCard,
} from "./SoftwareOSDetailsPage";

const mockRouter = createMockRouter();

describe("SummaryCard", () => {
  it("renders OS name, host count, and updated timestamp", () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);
    const twoDaysAgo = currentDate.toISOString();

    render(
      <SummaryCard
        osVersion={createMockOSVersion({ hosts_count: 123 })}
        countsUpdatedAt={twoDaysAgo}
        teamIdForApi={7}
      />
    );

    expect(screen.getByText("Mac OS X")).toBeInTheDocument();
    expect(screen.getByText(/123/)).toBeInTheDocument();
    expect(screen.getByText(/Updated 2 days ago/)).toBeInTheDocument();
  });
});

describe("VulnerabilitiesCard", () => {
  it("renders vulnerability table if platform supports vulns", () => {
    render(
      <VulnerabilitiesCard
        osVersion={createMockOSVersion()}
        isLoading={false}
        router={mockRouter}
        teamIdForApi={1}
      />
    );

    expect(screen.getByText("Vulnerabilities")).toBeInTheDocument();
    expect(screen.getByText("Detected")).toBeInTheDocument();
    expect(screen.getByText("CVE-2020-0001")).toBeInTheDocument();
    expect(screen.getByText("Unavailable")).toBeInTheDocument(); // No created_at date
  });

  it("renders 'not supported' empty state if platform doesn't support vulns", () => {
    render(
      <VulnerabilitiesCard
        osVersion={createMockOSVersion({
          name: "iPadOS 17.6",
          name_only: "iPadOS",
          version: "17.5",
          platform: "ipados",
        })}
        isLoading={false}
        router={mockRouter}
      />
    );

    expect(
      screen.getByText(
        /Vulnerabilities are not supported for this type of host/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText(/iPadOS/i)).toBeInTheDocument();
  });
});

describe("KernelsCard", () => {
  it("renders kernels table with correct data", () => {
    render(
      <KernelsCard
        osVersion={createMockLinuxOSVersion()}
        isLoading={false}
        router={mockRouter}
        teamIdForApi={2}
      />
    );

    expect(screen.getByText("Kernels")).toBeInTheDocument();
    expect(screen.getByText("35 items")).toBeInTheDocument();
    expect(screen.getAllByText("6.11.0-26.26~24.04.1").length).toBeGreaterThan(
      0
    );
    expect(screen.getByText("14 vulnerabilities")).toBeInTheDocument();
  });

  it("renders 'no kernels detected' if no data on kernels", () => {
    render(
      <KernelsCard
        osVersion={createMockLinuxOSVersion({ kernels: [] })}
        isLoading={false}
        router={mockRouter}
        teamIdForApi={2}
      />
    );

    expect(screen.getByText("Kernels")).toBeInTheDocument();
    expect(screen.getByText("No kernels detected")).toBeInTheDocument();
    expect(screen.getByText("Expecting to see kernels?")).toBeInTheDocument();
  });
});
