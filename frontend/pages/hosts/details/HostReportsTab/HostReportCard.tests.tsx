import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { IHostReport } from "services/entities/host_reports";

import HostReportCard from "./HostReportCard";

jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));

const mockedUseCheckTruncatedElement = useCheckTruncatedElement as jest.Mock;

const createReport = (firstResult: Record<string, string>): IHostReport => ({
  report_id: 1,
  name: "Test Report",
  description: "",
  last_fetched: "2024-01-01T00:00:00Z",
  first_result: firstResult,
  n_host_results: 1,
  report_clipped: false,
  store_results: true,
});

const renderCard = (firstResult: Record<string, string>) =>
  render(
    <HostReportCard
      report={createReport(firstResult)}
      hostName="test-host"
      onShowDetails={jest.fn()}
      onViewAllHosts={jest.fn()}
    />
  );

describe("HostReportCard", () => {
  beforeEach(() => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);
  });

  it("truncates the tooltip of a long result value", async () => {
    const user = userEvent.setup();
    const longValue = `${"a".repeat(400)}END`;
    renderCard({ big: longValue });

    await user.hover(screen.getByText(longValue));

    const tooltip = await screen.findByText("(truncated)");
    expect(tooltip).toBeInTheDocument();
    expect(screen.getAllByText(longValue)).toHaveLength(1);
  });

  it("shows the full value in the tooltip of a short result value", async () => {
    const user = userEvent.setup();
    renderCard({ col1: "short value" });

    await user.hover(screen.getByText("short value"));

    await waitFor(() =>
      expect(screen.getAllByText("short value")).toHaveLength(2)
    );
    expect(screen.queryByText("(truncated)")).not.toBeInTheDocument();
  });
});
