import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
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
        renderActionsDropdown={renderActionDropdown}
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
        renderActionsDropdown={renderActionDropdown}
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
        renderActionsDropdown={renderActionDropdown}
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
        renderActionsDropdown={renderActionDropdown}
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
        renderActionsDropdown={renderActionDropdown}
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
        renderActionsDropdown={renderActionDropdown}
      />
    );
    fireEvent.click(screen.getByText("Refetch"));
    expect(onRefetchHost).toHaveBeenCalled();
  });

  it("shows tooltip when host is offline", async () => {
    const { user } = renderWithSetup(
      <HostHeader
        summaryData={{ ...defaultSummaryData, status: "offline" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
      />
    );

    await user.hover(screen.getByText("Refetch"));

    expect(await screen.findByText(/an offline host/i)).toBeInTheDocument();
  });

  it("prioritises showing host status tooltips over offline tooltips on the refetch button", async () => {
    const { user } = renderWithSetup(
      <HostHeader
        summaryData={{ ...defaultSummaryData, status: "offline" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"locked" as HostMdmDeviceStatusUIState}
      />
    );

    await user.hover(screen.getByText("Refetch"));

    expect(await screen.findByText(/a locked host/i)).toBeInTheDocument();
  });

  it("renders device status tag and tooltip if hostMdmDeviceStatus is set", async () => {
    const { user } = renderWithSetup(
      <HostHeader
        summaryData={defaultSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"locked" as HostMdmDeviceStatusUIState}
      />
    );

    await user.hover(screen.getByText("Locked"));

    expect(await screen.findByText(/Host is locked/i)).toBeInTheDocument();
  });

  it("does NOT render the 'Locked' badge for Android hosts but DOES render Wipe pending / Wiped / Lock pending (#41683)", () => {
    // AMAPI does not deliver a "device-is-still-locked" notification — the user unlocks locally
    // with their PIN with no signal back to Fleet — so a "Locked" badge for Android would be
    // unreliable. Pending lock/wipe and Wiped are all derived from server-tracked Pub/Sub COMMAND
    // acks and are safe to display.
    const { rerender } = renderWithSetup(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"locked" as HostMdmDeviceStatusUIState}
      />
    );
    expect(screen.queryByText("Locked")).not.toBeInTheDocument();

    rerender(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiped" as HostMdmDeviceStatusUIState}
      />
    );
    expect(screen.getByText("Wiped")).toBeInTheDocument();
  });

  it("renders 'Unenroll pending' (not 'Wipe pending') for BYO Android during pending wipe (#41683)", () => {
    // BYO Android Unenroll fires an AMAPI WIPE under the hood, so the backend surfaces this as
    // hostMdmDeviceStatus="wiping". The label is overridden in HostHeader for BYO so the badge
    // matches the action the admin took (Unenroll), not the underlying mechanism.
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiping" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus="On (personal)"
      />
    );
    expect(screen.getByText("Unenroll pending")).toBeInTheDocument();
    expect(screen.queryByText("Wipe pending")).not.toBeInTheDocument();
  });

  it("renders 'Wipe pending' for COBO Android during pending wipe (#41683)", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiping" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus="On (automatic)"
      />
    );
    expect(screen.getByText("Wipe pending")).toBeInTheDocument();
    expect(screen.queryByText("Unenroll pending")).not.toBeInTheDocument();
  });

  it("renders 'Clear passcode pending' badge for Android (#41683)", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"clearing_passcode" as HostMdmDeviceStatusUIState}
      />
    );
    expect(screen.getByText("Clear passcode pending")).toBeInTheDocument();
  });
});
