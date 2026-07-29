import React from "react";
import { noop } from "lodash";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockHost from "__mocks__/hostMock";
import { createMockHostMdmData } from "__mocks__/mdmMock";

import { IHost } from "interfaces/host";
import VitalsModal from "./VitalsModal";
import { UNSUPPORTED_VITALS_BY_ENROLLMENT } from "./unsupportedVitalsByEnrollment";

const ALL_VITALS_LABELS = [
  "Accessibility settings",
  "App analytics",
  "Awaiting configuration",
  "Battery level",
  "Bluetooth MAC address",
  "Cellular technology",
  "Cloud backup enabled",
  "Data roaming",
  "Device locator service enabled",
  "Device properties attestation",
  "Diagnostic submission",
  "Do not disturb",
  "EAS device identifier",
  "iTunes Store account active",
  "iTunes Store account hash",
  "Last cloud backup",
  "Lost mode",
  "MDM options",
  "Model number",
  "Modem firmware version",
  "Network tethered",
  "Organization info",
  "Personal hotspot",
  "Push token",
  "Service subscriptions",
  "Supplemental build version",
  "Supplemental OS version extra",
  "UDID",
  "Wi-Fi MAC address",
];

const buildFullyPopulatedHost = (overrides?: Partial<IHost>): IHost =>
  createMockHost({
    platform: "ios",
    mdm: createMockHostMdmData({ enrollment_status: "On (manual)" }),
    udid: "00008030-000000000000000",
    model_number: "MU123",
    modem_firmware_version: "1.0",
    supplemental_build_version: "21A5326a",
    supplemental_os_version_extra: "(a)",
    bluetooth_mac: "AA:BB:CC:DD:EE:FF",
    wifi_mac: "11:22:33:44:55:66",
    eas_device_identifier: "eas-id-123",
    itunes_store_account_hash: "abc123hash",
    push_token: "cHVzaC10b2tlbg==",
    battery_level: 0.87,
    cellular_technology: "GSM",
    app_analytics_enabled: true,
    awaiting_configuration: false,
    data_roaming_enabled: false,
    diagnostic_submission_enabled: true,
    is_cloud_backup_enabled: true,
    is_device_locator_service_enabled: true,
    is_do_not_disturb_in_effect: false,
    is_mdm_lost_mode_enabled: false,
    is_network_tethered: false,
    itunes_store_account_is_active: true,
    personal_hotspot_enabled: false,
    last_cloud_backup_date: "2022-01-02T12:00:00Z",
    accessibility_settings: { zoom_enabled: true },
    organization_info: { organization_name: "Fleet Device Management" },
    mdm_options: { bootstrap_token_allowed: true },
    device_properties_attestation: ["AAAA"],
    service_subscriptions: [{ slot: "primary" }],
    ...overrides,
  });

describe("VitalsModal component", () => {
  it("renders all 29 new vitals fields, alphabetically sorted", () => {
    const host = buildFullyPopulatedHost();

    const { container } = render(<VitalsModal host={host} onExit={noop} />);

    const renderedLabels = Array.from(
      container.querySelectorAll(".vitals-modal .data-set > dt")
    ).map((el) => el.textContent);

    ALL_VITALS_LABELS.forEach((label) => {
      expect(renderedLabels).toContain(label);
    });
    expect(renderedLabels).toEqual(
      [...ALL_VITALS_LABELS].sort((a, b) => a.localeCompare(b))
    );
  });

  it("renders scalar and boolean vital values", () => {
    const host = buildFullyPopulatedHost();

    render(<VitalsModal host={host} onExit={noop} />);

    expect(screen.getByText("00008030-000000000000000")).toBeInTheDocument();
    expect(screen.getByText("87%")).toBeInTheDocument();
    // The API maps Apple's integer code to a label; the modal renders it as-is.
    expect(screen.getByText("GSM")).toBeInTheDocument();
    // app_analytics_enabled: true -> "True"; awaiting_configuration: false -> "False"
    expect(screen.getAllByText("True").length).toBeGreaterThan(0);
    expect(screen.getAllByText("False").length).toBeGreaterThan(0);
  });

  it("renders service subscriptions and device properties attestation as one line per entry, not a JSON preview", () => {
    const host = buildFullyPopulatedHost({
      service_subscriptions: [{ slot: "primary" }, { slot: "secondary" }],
      device_properties_attestation: ["Y2VydC1vbmU=", "Y2VydC10d28="],
    });

    const { container } = render(<VitalsModal host={host} onExit={noop} />);

    const findValueCell = (label: string) =>
      Array.from(container.querySelectorAll(".data-set"))
        .find((dataSet) => dataSet.querySelector("dt")?.textContent === label)
        ?.querySelector("dd");

    const subscriptionsLines = findValueCell(
      "Service subscriptions"
    )?.querySelectorAll(".vitals-modal__lines > div");
    expect(
      Array.from(subscriptionsLines ?? []).map((el) => el.textContent)
    ).toEqual(["primary", "secondary"]);

    const attestationLines = findValueCell(
      "Device properties attestation"
    )?.querySelectorAll(".vitals-modal__lines > div");
    expect(
      Array.from(attestationLines ?? []).map((el) => el.textContent)
    ).toEqual(["Y2VydC1vbmU=", "Y2VydC10d28="]);

    // accessibility_settings, organization_info, mdm_options only
    expect(container.querySelectorAll("pre")).toHaveLength(3);
  });

  it("renders 'None' for a null field not marked unsupported", () => {
    const host = buildFullyPopulatedHost({ model_number: undefined });

    const { container } = render(<VitalsModal host={host} onExit={noop} />);

    const modelNumberValue = Array.from(container.querySelectorAll(".data-set"))
      .find(
        (dataSet) => dataSet.querySelector("dt")?.textContent === "Model number"
      )
      ?.querySelector("dd")?.textContent;

    expect(modelNumberValue).toBe("None");
  });

  describe("Not supported treatment", () => {
    const originalTable = JSON.parse(
      JSON.stringify(UNSUPPORTED_VITALS_BY_ENROLLMENT)
    );

    afterEach(() => {
      (Object.keys(UNSUPPORTED_VITALS_BY_ENROLLMENT) as Array<
        keyof typeof UNSUPPORTED_VITALS_BY_ENROLLMENT
      >).forEach((enrollmentStatus) => {
        UNSUPPORTED_VITALS_BY_ENROLLMENT[enrollmentStatus] = [
          ...originalTable[enrollmentStatus],
        ];
      });
    });

    it("renders the tooltip-wrapped 'Not supported' treatment for a vital marked unsupported for the host's enrollment method, even when the API returned a non-null value", async () => {
      UNSUPPORTED_VITALS_BY_ENROLLMENT["On (manual)"] = ["udid"];
      const host = buildFullyPopulatedHost({
        udid: "00008030-should-not-show",
      });
      const customRender = createCustomRenderer({});

      const { user } = customRender(<VitalsModal host={host} onExit={noop} />);

      expect(
        screen.queryByText("00008030-should-not-show")
      ).not.toBeInTheDocument();
      const notSupportedText = screen.getByText("Not supported");
      expect(notSupportedText).toBeInTheDocument();

      await user.hover(notSupportedText);
      await waitFor(() => {
        expect(
          screen.getByText(
            /This property isn't supported for this device's enrollment method\./i
          )
        ).toBeInTheDocument();
      });
    });

    it("does not apply the 'Not supported' treatment to vitals for a different enrollment method", () => {
      UNSUPPORTED_VITALS_BY_ENROLLMENT["On (automatic)"] = ["udid"];
      const host = buildFullyPopulatedHost({
        mdm: createMockHostMdmData({ enrollment_status: "On (manual)" }),
        udid: "00008030-000000000000000",
      });

      render(<VitalsModal host={host} onExit={noop} />);

      expect(screen.getByText("00008030-000000000000000")).toBeInTheDocument();
      expect(screen.queryByText("Not supported")).not.toBeInTheDocument();
    });
  });

  it("calls onExit when the Done button is clicked", async () => {
    const host = buildFullyPopulatedHost();
    const onExit = jest.fn();
    const customRender = createCustomRenderer({});

    const { user } = customRender(<VitalsModal host={host} onExit={onExit} />);

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(onExit).toHaveBeenCalled();
  });
});
