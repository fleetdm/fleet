import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";
import { internationalTimeFormat } from "utilities/helpers";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
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
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("My device")).toBeInTheDocument();
    expect(screen.getByText(/unavailable/i)).toBeInTheDocument();
  });
  it("renders a disabled refetch button for Android", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByRole("button", { name: /refetch/i })).toBeDisabled();
  });

  it("shows a tooltip on the disabled refetch button explaining why Android hosts can't be refetched", async () => {
    const { user } = renderWithSetup(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmEnrollmentStatus={null}
      />
    );

    await user.hover(screen.getByText("Refetch"));

    expect(await screen.findByText(/there's no manual/i)).toBeInTheDocument();
  });

  it("disables refetch button when host is offline", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, status: "offline" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
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
        hostMdmEnrollmentStatus={null}
      />
    );

    await user.hover(screen.getByText("Locked"));

    expect(await screen.findByText(/Host is locked/i)).toBeInTheDocument();
  });

  it("renders wipe status tags for Linux hosts", () => {
    const linuxSummaryData = { ...defaultSummaryData, platform: "ubuntu" };

    const { rerender } = render(
      <HostHeader
        summaryData={linuxSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiping" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("Wipe pending")).toBeInTheDocument();

    rerender(
      <HostHeader
        summaryData={linuxSummaryData}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiped" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("Wiped")).toBeInTheDocument();
  });

  it("renders 'Lock pending' and 'Wiped' badges for Android hosts", () => {
    const { rerender } = renderWithSetup(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"locking" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("Lock pending")).toBeInTheDocument();

    rerender(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"wiped" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("Wiped")).toBeInTheDocument();
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
  });

  it("renders 'Clear passcode pending' badge for Android (#41683)", () => {
    render(
      <HostHeader
        summaryData={{ ...defaultSummaryData, platform: "android" }}
        showRefetchSpinner={false}
        onRefetchHost={jest.fn()}
        renderActionsDropdown={renderActionDropdown}
        hostMdmDeviceStatus={"clearing_passcode" as HostMdmDeviceStatusUIState}
        hostMdmEnrollmentStatus={null}
      />
    );
    expect(screen.getByText("Clear passcode pending")).toBeInTheDocument();
  });

  describe("last MDM check-in", () => {
    const LAST_CHECK_IN = "2024-04-27T11:00:00Z";

    it("shows the last fetched and last MDM check-in times in a tooltip when the host has checked in", async () => {
      const { user } = renderWithSetup(
        <HostHeader
          summaryData={{
            ...defaultSummaryData,
            last_mdm_checked_in_at: LAST_CHECK_IN,
          }}
          showRefetchSpinner={false}
          onRefetchHost={jest.fn()}
          renderActionsDropdown={renderActionDropdown}
          hostMdmEnrollmentStatus="On (manual)"
        />
      );

      await user.hover(screen.getByText(/Last fetched/i));

      expect(await screen.findByText("Last MDM check-in:")).toBeInTheDocument();
      expect(
        screen.getByText(internationalTimeFormat(new Date(LAST_CHECK_IN)))
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          internationalTimeFormat(
            new Date(defaultSummaryData.detail_updated_at)
          )
        )
      ).toBeInTheDocument();
    });

    it("does not show the check-in tooltip when the host has never checked in", async () => {
      // normalizeEmptyValues turns an empty check-in timestamp into "---"
      const { user } = renderWithSetup(
        <HostHeader
          summaryData={{
            ...defaultSummaryData,
            last_mdm_checked_in_at: DEFAULT_EMPTY_CELL_VALUE,
          }}
          showRefetchSpinner={false}
          onRefetchHost={jest.fn()}
          renderActionsDropdown={renderActionDropdown}
          hostMdmEnrollmentStatus="On (manual)"
        />
      );

      await user.hover(screen.getByText(/Last fetched/i));

      expect(screen.queryByText("Last MDM check-in:")).not.toBeInTheDocument();
    });

    it("shows the MDM check-in row even when last fetched is unavailable", async () => {
      const { user } = renderWithSetup(
        <HostHeader
          summaryData={{
            ...defaultSummaryData,
            detail_updated_at: undefined,
            policy_updated_at: undefined,
            last_mdm_checked_in_at: LAST_CHECK_IN,
          }}
          showRefetchSpinner={false}
          onRefetchHost={jest.fn()}
          renderActionsDropdown={renderActionDropdown}
          hostMdmEnrollmentStatus="On (manual)"
        />
      );

      await user.hover(screen.getByText(/Last fetched/i));

      expect(await screen.findByText("Last MDM check-in:")).toBeInTheDocument();
      expect(
        screen.getByText(internationalTimeFormat(new Date(LAST_CHECK_IN)))
      ).toBeInTheDocument();
      // Empty bold next to "Last fetched:" would look broken; render an
      // explicit fallback instead.
      expect(screen.getByText("unavailable")).toBeInTheDocument();
    });

    it("uses the more recent of detail_updated_at and policy_updated_at as 'Last fetched' (#51820)", async () => {
      const DETAIL_UPDATED_AT = "2024-04-27T12:00:00Z";
      const POLICY_UPDATED_AT = "2024-04-27T13:30:00Z"; // 90 min newer
      const { user } = renderWithSetup(
        <HostHeader
          summaryData={{
            ...defaultSummaryData,
            detail_updated_at: DETAIL_UPDATED_AT,
            policy_updated_at: POLICY_UPDATED_AT,
            last_mdm_checked_in_at: LAST_CHECK_IN,
          }}
          showRefetchSpinner={false}
          onRefetchHost={jest.fn()}
          renderActionsDropdown={renderActionDropdown}
          hostMdmEnrollmentStatus="On (manual)"
        />
      );

      await user.hover(screen.getByText(/Last fetched/i));

      expect(
        await screen.findByText(
          internationalTimeFormat(new Date(POLICY_UPDATED_AT))
        )
      ).toBeInTheDocument();
      expect(
        screen.queryByText(internationalTimeFormat(new Date(DETAIL_UPDATED_AT)))
      ).not.toBeInTheDocument();
    });

    it("renders last fetched without a check-in tooltip when the platform reports no check-in field at all", async () => {
      // lodash `pick` drops the key entirely for platforms whose API response
      // omits it, so the header gets `undefined` rather than "---".
      const { user } = renderWithSetup(
        <HostHeader
          summaryData={defaultSummaryData}
          showRefetchSpinner={false}
          onRefetchHost={jest.fn()}
          renderActionsDropdown={renderActionDropdown}
          hostMdmEnrollmentStatus={null}
        />
      );

      expect(screen.getByText(/Last fetched/i)).toBeInTheDocument();

      await user.hover(screen.getByText(/Last fetched/i));

      expect(screen.queryByText("Last MDM check-in:")).not.toBeInTheDocument();
    });
  });
});
