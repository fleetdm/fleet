import React from "react";
import { noop } from "lodash";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockHost from "__mocks__/hostMock";
import { createMockHostMdmData } from "__mocks__/mdmMock";

import { IHost } from "interfaces/host";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import VitalsModal from "./VitalsModal";

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

/** Finds a vital row's value cell by its display label. */
const findValueCell = (container: HTMLElement, label: string) =>
  Array.from(container.querySelectorAll(".data-set"))
    .find((dataSet) => dataSet.querySelector("dt")?.textContent === label)
    ?.querySelector("dd");

describe("VitalsModal component", () => {
  it("renders the 29 iOS/iPadOS vitals alongside the pre-existing ones, in one alphabetical list", () => {
    const host = buildFullyPopulatedHost();

    const { container } = render(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    const renderedLabels = Array.from(
      container.querySelectorAll(".vitals-modal .data-set > dt")
    ).map((el) => el.textContent ?? "");

    // Every iOS/iPadOS-only vital.
    ALL_VITALS_LABELS.forEach((label) => {
      expect(renderedLabels).toContain(label);
    });

    // Plus the pre-existing vitals, which the iOS/iPadOS card now trims away —
    // the modal is the only place they remain visible.
    ["Added to Fleet", "Hardware model", "Operating system"].forEach(
      (label) => {
        expect(renderedLabels).toContain(label);
      }
    );

    // Merged into a single alphabetical ordering rather than appended.
    expect(renderedLabels).toEqual(
      [...renderedLabels].sort((a, b) => a.localeCompare(b))
    );
  });

  it("renders scalar and boolean vital values", () => {
    const host = buildFullyPopulatedHost();

    render(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    expect(screen.getByText("00008030-000000000000000")).toBeInTheDocument();
    expect(screen.getByText("87%")).toBeInTheDocument();
    // The API maps Apple's integer code to a label; the modal renders it as-is.
    expect(screen.getByText("GSM")).toBeInTheDocument();
    // app_analytics_enabled: true -> "True"; awaiting_configuration: false -> "False"
    expect(screen.getAllByText("True").length).toBeGreaterThan(0);
    expect(screen.getAllByText("False").length).toBeGreaterThan(0);
  });

  it("treats a negative battery level as unknown, since Apple reports -1 when it can't determine one", () => {
    const host = buildFullyPopulatedHost({ battery_level: -1 });

    const { container } = render(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    expect(findValueCell(container, "Battery level")?.textContent).toBe(
      DEFAULT_EMPTY_CELL_VALUE
    );
    expect(screen.queryByText("-100%")).not.toBeInTheDocument();
  });

  it("renders the attestation certificate chain as one line per certificate", () => {
    const host = buildFullyPopulatedHost({
      device_properties_attestation: ["Y2VydC1vbmU=", "Y2VydC10d28="],
    });

    const { container } = render(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    const lines = findValueCell(
      container,
      "Device properties attestation"
    )?.querySelectorAll(".vitals-modal__lines > div");

    expect(Array.from(lines ?? []).map((el) => el.textContent)).toEqual([
      "Y2VydC1vbmU=",
      "Y2VydC10d28=",
    ]);

    // Nested dicts render as label/value lines, so no code block anywhere.
    expect(container.querySelectorAll("pre")).toHaveLength(0);
  });

  it("renders one numbered subscription per SIM, revealing that subscription's values on hover", async () => {
    const host = buildFullyPopulatedHost({
      service_subscriptions: [
        {
          slot: "CTSubscriptionSlotOne",
          label: "Principal",
          phone_number: "+5491100000000",
          is_roaming: false,
        },
        // A dual-SIM device with an empty eSIM slot reports only a couple of
        // fields for it.
        { slot: "CTSubscriptionSlotTwo", eid: "8904903200740888" },
      ],
    });
    const customRender = createCustomRenderer({});

    const { user, container } = customRender(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    const cell = findValueCell(container, "Service subscriptions");
    expect(cell?.textContent).toContain("Subscription 1");
    expect(cell?.textContent).toContain("Subscription 2");

    await user.hover(screen.getByText("Subscription 1"));

    await waitFor(() => {
      expect(screen.getByText("Phone number:").tagName).toBe("B");
    });
    expect(screen.getByText(/\+5491100000000/)).toBeInTheDocument();
    expect(screen.getByText("Label:")).toBeInTheDocument();
    expect(screen.getByText("Roaming:")).toBeInTheDocument();

    await user.hover(screen.getByText("Subscription 2"));
    await waitFor(() => {
      expect(screen.getByText(/8904903200740888/)).toBeInTheDocument();
    });
    expect(screen.queryAllByText("Phone number:")).toHaveLength(1);
  });

  // Enrollment ID's tooltip is defined in buildHostVitals, so this guards
  // that the card and the modal really do share one implementation.
  it("keeps Enrollment ID's existing tooltip, which comes from the shared vitals builder", async () => {
    const host = buildFullyPopulatedHost({
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    await user.hover(screen.getByText("Enrollment ID"));

    await waitFor(() => {
      expect(
        screen.getByText(/Enrollment ID is a unique identifier for personal/i)
      ).toBeInTheDocument();
    });
  });

  describe("Nested-dict vitals", () => {
    const findNestedPairs = (container: HTMLElement, label: string) => {
      const nested = Array.from(container.querySelectorAll(".data-set"))
        .find((dataSet) => dataSet.querySelector("dt")?.textContent === label)
        ?.querySelector(".vitals-modal__nested");

      return Array.from(nested?.children ?? []).map((el) => el.textContent);
    };

    it("renders each present sub-key as a label/value pair, with hand-written labels rather than raw API keys", () => {
      const host = buildFullyPopulatedHost({
        organization_info: {
          organization_name: "Fleet Device Management",
          organization_email: "support@example.com",
        },
      });

      const { container } = render(
        <VitalsModal
          host={host}
          vitalsData={host}
          mdm={host.mdm}
          onExit={noop}
        />
      );

      expect(findNestedPairs(container, "Organization info")).toEqual([
        "Email:",
        "support@example.com",
        "Name:",
        "Fleet Device Management",
      ]);
      // Raw snake_case API keys must never reach the UI.
      expect(screen.queryByText("organization_name")).not.toBeInTheDocument();
    });

    it("keeps a false sub-key but drops absent ones", () => {
      const host = buildFullyPopulatedHost({
        accessibility_settings: {
          zoom_enabled: false,
          text_size: 5,
        },
      });

      const { container } = render(
        <VitalsModal
          host={host}
          vitalsData={host}
          mdm={host.mdm}
          onExit={noop}
        />
      );

      expect(findNestedPairs(container, "Accessibility settings")).toEqual([
        "Text size:",
        "5",
        "Zoom:",
        "False",
      ]);
    });

    it("falls back to the empty value when every sub-key is absent (Apple's empty MDMOptions dict)", () => {
      const host = buildFullyPopulatedHost({ mdm_options: {} });

      const { container } = render(
        <VitalsModal
          host={host}
          vitalsData={host}
          mdm={host.mdm}
          onExit={noop}
        />
      );

      const mdmOptionsValue = Array.from(
        container.querySelectorAll(".data-set")
      )
        .find(
          (dataSet) =>
            dataSet.querySelector("dt")?.textContent === "MDM options"
        )
        ?.querySelector("dd")?.textContent;

      expect(mdmOptionsValue).toBe(DEFAULT_EMPTY_CELL_VALUE);
    });
  });

  it("renders the empty-cell placeholder for a null field not marked unsupported", () => {
    const host = buildFullyPopulatedHost({ model_number: undefined });

    const { container } = render(
      <VitalsModal host={host} vitalsData={host} mdm={host.mdm} onExit={noop} />
    );

    const modelNumberValue = Array.from(container.querySelectorAll(".data-set"))
      .find(
        (dataSet) => dataSet.querySelector("dt")?.textContent === "Model number"
      )
      ?.querySelector("dd")?.textContent;

    expect(modelNumberValue).toBe(DEFAULT_EMPTY_CELL_VALUE);
  });

  it("calls onExit when the Done button is clicked", async () => {
    const host = buildFullyPopulatedHost();
    const onExit = jest.fn();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <VitalsModal
        host={host}
        vitalsData={host}
        mdm={host.mdm}
        onExit={onExit}
      />
    );

    await user.click(screen.getByRole("button", { name: "Done" }));

    expect(onExit).toHaveBeenCalled();
  });

  it("calls onExit when Escape is pressed", async () => {
    const host = buildFullyPopulatedHost();
    const onExit = jest.fn();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <VitalsModal
        host={host}
        vitalsData={host}
        mdm={host.mdm}
        onExit={onExit}
      />
    );

    await user.keyboard("{Escape}");

    // Modal defers onExit until its close animation finishes, so this can't be
    // asserted synchronously the way the Done button (a direct onExit) can.
    await waitFor(() => expect(onExit).toHaveBeenCalledTimes(1));
  });
});
