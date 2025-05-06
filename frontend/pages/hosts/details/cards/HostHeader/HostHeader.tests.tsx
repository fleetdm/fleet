import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import HostHeader from "./HostHeader";
import { HostMdmDeviceStatusUIState } from "../../helpers";

const renderActionDropdown = jest.fn(() => <div data-testid="dropdown" />);

const defaultSummaryData = {
  platform: "darwin",
  status: "online",
  display_name: "Test Host",
  detail_updated_at: "2024-04-27T12:00:00Z",
};

describe("HostHeader", () => {
  it("renders host display name and last fetched", () => {
    render(
      <HostHeader
        summaryData={defaultSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
      />
    );
    expect(screen.getByText("Test Host")).toBeInTheDocument();
    expect(screen.getByText(/Last fetched/i)).toBeInTheDocument();
  });

  it("renders 'My device' when deviceUser is true and  unavailable when no last fetched date", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, detail_updated_at: undefined }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
        deviceUser
      />
    );
    expect(screen.getByText("My device")).toBeInTheDocument();
    expect(screen.getByText(/unavailable/i)).toBeInTheDocument();
  });
  it("does not render refetch button for Android", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
      />
    );
    expect(screen.queryByText("Refetch")).not.toBeInTheDocument();
  });

  it("disables refetch button when host is offline", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, status: "offline" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
      />
    );
    const refetchButton = screen.getByRole("button", { name: /refetch/i });
    expect(refetchButton).toBeDisabled();
  });

  it("shows refetch spinner text when fetching", () => {
    render(
      <HostHeader
        summaryData={defaultSummaryData}
        showRefetchSpinner
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
      />
    );
    expect(screen.getByText(/Fetching fresh vitals/i)).toBeInTheDocument();
  });

  it("calls onRefetchHost when refetch button is clicked", () => {
    const onRefetchHost = jest.fn();
    render(
      <HostHeader
        summaryData={defaultSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={onRefetchHost}
        renderActionDropdown={renderActionDropdown}
      />
    );
    fireEvent.click(screen.getByText("Refetch"));
    expect(onRefetchHost).toHaveBeenCalled();
  });

  it("shows tooltip when host is offline", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, status: "offline" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
      />
    );
    expect(screen.getByText(/an offline host/i)).toBeInTheDocument();
  });

  it("renders device status tag and tooltip if hostMdmDeviceStatus is set", () => {
    render(
      <HostHeader
        summaryData={defaultSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"locked" as HostMdmDeviceStatusUIState}
      />
    );
    expect(screen.getByText(/a locked host/i)).toBeInTheDocument();
  });
});
