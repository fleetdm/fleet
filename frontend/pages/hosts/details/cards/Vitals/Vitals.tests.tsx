import React from "react";
import { render, screen, waitFor, within } from "@testing-library/react";
import { pick } from "lodash";
import { createCustomRenderer } from "test/test-utils";

import createMockHost, { createMockHostGeolocation } from "__mocks__/hostMock";
import { IHost } from "interfaces/host";
import { createMockHostMdmData } from "__mocks__/mdmMock";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import { HostPlatform } from "interfaces/platform";
import { IHostCustomVital } from "interfaces/custom_host_vitals";
import {
  DEFAULT_EMPTY_CELL_VALUE,
  HOST_VITALS_DATA,
} from "utilities/constants";
import { normalizeEmptyValues } from "utilities/helpers";
import Vitals from "./Vitals";

describe("Vitals Card component", () => {
  it("renders the device Hardware model and Serial number for Android hosts that were not enrolled in MDM personally", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "1234567890",
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("Pixel 6")).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getByText("1234567890")).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders device Hardware model and Enrollment ID for Android hosts enrolled in MDM personally", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("Pixel 6")).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("keeps Enrollment ID for a personally enrolled Android host after it unenrolls", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "Off",
        is_personal_enrollment: true,
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
  });

  it("renders Serial number for an unenrolled Android host that was not personally enrolled", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "1234567890",
      mdm: createMockHostMdmData({
        enrollment_status: "Off",
        is_personal_enrollment: false,
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getByText("1234567890")).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
  });

  it("tells the user the Enrollment ID is stable for personal Android hosts", async () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "Off",
        is_personal_enrollment: true,
      }),
    });
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals vitalsData={mockHost} mdm={mockHost.mdm} />
    );

    await user.hover(screen.getByText("Enrollment ID"));

    await waitFor(() => {
      expect(
        screen.getByText(
          /For personal \(BYOD\) Android hosts, this enrollment ID is stable across unenroll\/re-enroll cycles\./i
        )
      ).toBeInTheDocument();
    });
  });

  it("renders Enrollment ID and Hardware model for personally enrolled iOS hosts", () => {
    const mockHost = createMockHost({
      platform: "ios",
      hardware_model: "iPhone12,1",
      hardware_marketing_name: "iPhone 11",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("iPhone 11")).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Enrollment ID and Hardware model for personally enrolled iPad hosts", () => {
    const mockHost = createMockHost({
      platform: "ipados",
      hardware_model: "iPad14,5",
      hardware_marketing_name: "iPad Pro 12.9-inch (6th generation) Wi-Fi",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(
      screen.getByText("iPad Pro 12.9-inch (6th generation) Wi-Fi")
    ).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Serial number and Hardware model for non-personally enrolled iOS hosts", () => {
    const mockHost = createMockHost({
      platform: "ios",
      hardware_model: "iPhone12,1",
      hardware_marketing_name: "iPhone 11",
      hardware_serial: "123-456-789",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("iPhone 11")).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("123-456-789")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Enrollment ID and Hardware model for non-personally enrolled iPad hosts", () => {
    const mockHost = createMockHost({
      platform: "ipados",
      hardware_model: "iPad14,5",
      hardware_marketing_name: "iPad Pro 12.9-inch (6th generation) Wi-Fi",
      hardware_serial: "123-456-789",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (automatic)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(
      screen.getByText("iPad Pro 12.9-inch (6th generation) Wi-Fi")
    ).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("123-456-789")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and EnrollmentID for all non android and ios/ipad hosts that have enrolled their personal mdm devices", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBookPro18,1",
      hardware_marketing_name: "MacBook Pro (16-inch, 2021)",
      hardware_serial: "",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro (16-inch, 2021)")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and Serial number for all non android and ios/ipad hosts that have enrolled not enrolled in MDM", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBookPro18,1",
      hardware_marketing_name: "MacBook Pro (16-inch, 2021)",
      hardware_serial: "test-serial-number",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: undefined,
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro (16-inch, 2021)")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("test-serial-number")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and Serial number for all non android and ios/ipad hosts that have manually enrolled in MDM", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBookPro18,1",
      hardware_marketing_name: "MacBook Pro (16-inch, 2021)",
      hardware_serial: "test-serial-number",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro (16-inch, 2021)")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("test-serial-number")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and Serial number for all non android and ios/ipad hosts that have automatically enrolled in MDM", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBookPro18,1",
      hardware_marketing_name: "MacBook Pro (16-inch, 2021)",
      hardware_serial: "test-serial-number",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (automatic)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro (16-inch, 2021)")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("test-serial-number")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
  });
});

describe("Location vital", () => {
  // ADE = iOS/iPadOS host with mdm.enrollment_status === "On (automatic)";
  // matches the definition in Vitals.tsx.
  const renderLocationVital = ({
    ade = false,
    withToggle = false,
    hostOverrides,
  }: {
    ade?: boolean;
    withToggle?: boolean;
    hostOverrides?: Parameters<typeof createMockHost>[0];
  } = {}) => {
    const baseOverrides = ade
      ? {
          platform: "ios" as const,
          mdm: createMockHostMdmData({ enrollment_status: "On (automatic)" }),
        }
      : {
          platform: "darwin" as const,
          geolocation: createMockHostGeolocation(),
        };

    const mockHost = createMockHost({ ...baseOverrides, ...hostOverrides });
    const toggleLocationModal = withToggle ? jest.fn() : undefined;

    const utils = render(
      <Vitals
        vitalsData={mockHost}
        mdm={mockHost.mdm}
        toggleLocationModal={toggleLocationModal}
      />
    );

    return { ...utils, toggleLocationModal };
  };

  it("renders city/country as a clickable button when toggleLocationModal is provided", () => {
    renderLocationVital({ withToggle: true });

    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Minneapolis, US" })
    ).toBeInTheDocument();
  });

  it("invokes toggleLocationModal when the city/country button is clicked", () => {
    const { toggleLocationModal } = renderLocationVital({ withToggle: true });

    screen.getByRole("button", { name: "Minneapolis, US" }).click();

    expect(toggleLocationModal).toHaveBeenCalledTimes(1);
  });

  it("renders city/country as plain text when toggleLocationModal is not provided (e.g., My device page)", () => {
    renderLocationVital();

    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.getByText("Minneapolis, US")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Minneapolis, US" })
    ).not.toBeInTheDocument();
  });

  it("backfills the Location row onto an iOS/iPadOS card that is short a priority vital", () => {
    // Dropping Timezone leaves only 7 priority vitals, so the card has one
    // slot left for the backfill to fill.
    renderLocationVital({
      ade: true,
      withToggle: true,
      hostOverrides: { timezone: null },
    });

    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Show location" })
    ).toBeInTheDocument();
  });

  it("hides the Location row for ADE-enrolled iDevices when toggleLocationModal is not provided", () => {
    renderLocationVital({ ade: true });

    expect(screen.queryByText("Location")).not.toBeInTheDocument();
    expect(screen.queryByText("Show location")).not.toBeInTheDocument();
  });

  it("hides the Location row when the host has no geolocation", () => {
    renderLocationVital({ hostOverrides: { geolocation: undefined } });

    expect(screen.queryByText("Location")).not.toBeInTheDocument();
  });

  it("renders an empty Location value when geolocation is present but city/country are empty strings", () => {
    renderLocationVital({
      hostOverrides: {
        geolocation: createMockHostGeolocation({
          city_name: "",
          country_iso: "",
        }),
      },
    });

    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});

describe("MDM status vital", () => {
  const renderMDMStatusVital = ({ withToggle = false } = {}) => {
    const mockHost = createMockHost({
      platform: "darwin",
      mdm: createMockHostMdmData({ enrollment_status: "On (manual)" }),
    });
    const toggleMDMStatusModal = withToggle ? jest.fn() : undefined;

    const utils = render(
      <Vitals
        vitalsData={mockHost}
        mdm={mockHost.mdm}
        toggleMDMStatusModal={toggleMDMStatusModal}
      />
    );

    return { ...utils, toggleMDMStatusModal };
  };

  it("renders the MDM status as a clickable button when toggleMDMStatusModal is provided", () => {
    renderMDMStatusVital({ withToggle: true });

    expect(screen.getByText("MDM status")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "On (manual)" })
    ).toBeInTheDocument();
  });

  it("invokes toggleMDMStatusModal when the status button is clicked", () => {
    const { toggleMDMStatusModal } = renderMDMStatusVital({ withToggle: true });

    screen.getByRole("button", { name: "On (manual)" }).click();

    expect(toggleMDMStatusModal).toHaveBeenCalledTimes(1);
  });

  it("renders the MDM status as plain text when toggleMDMStatusModal is not provided (e.g., My device page)", () => {
    renderMDMStatusVital();

    expect(screen.getByText("MDM status")).toBeInTheDocument();
    expect(screen.getByText("On (manual)")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "On (manual)" })
    ).not.toBeInTheDocument();
  });
});

describe("View all vitals button", () => {
  const renderVitalsCard = ({
    platform = "ios",
    withToggle = false,
    enrollmentStatus,
  }: {
    platform?: HostPlatform;
    withToggle?: boolean;
    enrollmentStatus?: MdmEnrollmentStatus;
  } = {}) => {
    const mockHost = createMockHost({
      platform,
      ...(enrollmentStatus
        ? {
            mdm: createMockHostMdmData({ enrollment_status: enrollmentStatus }),
          }
        : {}),
    });
    const toggleVitalsModal = withToggle ? jest.fn() : undefined;

    const utils = render(
      <Vitals
        vitalsData={mockHost}
        mdm={mockHost.mdm}
        toggleVitalsModal={toggleVitalsModal}
      />
    );

    return { ...utils, toggleVitalsModal };
  };

  it("renders a 'View all' button for iOS hosts when toggleVitalsModal is provided", () => {
    renderVitalsCard({ platform: "ios", withToggle: true });

    expect(
      screen.getByRole("button", { name: "View all" })
    ).toBeInTheDocument();
  });

  it("renders a 'View all' button for iPadOS hosts when toggleVitalsModal is provided", () => {
    renderVitalsCard({ platform: "ipados", withToggle: true });

    expect(
      screen.getByRole("button", { name: "View all" })
    ).toBeInTheDocument();
  });

  it("invokes toggleVitalsModal when the 'View all' button is clicked", () => {
    const { toggleVitalsModal } = renderVitalsCard({
      platform: "ios",
      withToggle: true,
    });

    screen.getByRole("button", { name: "View all" }).click();

    expect(toggleVitalsModal).toHaveBeenCalledTimes(1);
  });

  it("does not render the button for iOS hosts when toggleVitalsModal is not provided (e.g., My device page)", () => {
    renderVitalsCard({ platform: "ios" });

    expect(
      screen.queryByRole("button", { name: "View all" })
    ).not.toBeInTheDocument();
  });

  it("does not render the button for non-iOS/iPadOS hosts even when toggleVitalsModal is provided", () => {
    renderVitalsCard({ platform: "darwin", withToggle: true });

    expect(
      screen.queryByRole("button", { name: "View all" })
    ).not.toBeInTheDocument();
  });

  it("does not render the button for a personal (BYOD) iOS host even when toggleVitalsModal is provided", () => {
    renderVitalsCard({
      platform: "ios",
      withToggle: true,
      enrollmentStatus: "On (manual - personal)",
    });

    expect(
      screen.queryByRole("button", { name: "View all" })
    ).not.toBeInTheDocument();
  });
});

describe("Card vitals cap", () => {
  const IOS_CARD_VITAL_ROWS = 2;
  const FALLBACK_COLUMN_COUNT = 6;
  // jsdom does no layout, so getComputedStyle can't resolve the grid's track
  // list and the card falls back to its widest-breakpoint column count.
  const CAP = FALLBACK_COLUMN_COUNT * IOS_CARD_VITAL_ROWS;

  const getRenderedVitals = (container: HTMLElement) =>
    Array.from(
      container.querySelectorAll(".vitals-card__info-grid > .data-set > dt")
    ).map((el) => el.textContent);

  const isAlphabetical = (labels: (string | null)[]) =>
    labels.every(
      (label, i) =>
        i === 0 || (labels[i - 1] ?? "").localeCompare(label ?? "") <= 0
    );

  /** Mirrors production: HostDetailsPage/DeviceUserPage both hand the card
   * normalizeEmptyValues(pick(host, HOST_VITALS_DATA)), which replaces empty
   * values with DEFAULT_EMPTY_CELL_VALUE rather than leaving them null. */
  const renderCard = (
    platform: HostPlatform,
    {
      customHostVitals,
      withModal = true,
      enrollmentStatus = "On (manual)",
    }: {
      customHostVitals?: IHostCustomVital[];
      withModal?: boolean;
      enrollmentStatus?: MdmEnrollmentStatus;
    } = {}
  ) => {
    const mockHost = createMockHost({
      platform,
      hardware_serial: "test-serial",
      timezone: "America/Argentina/Buenos_Aires",
      memory: 8589934592,
      cpu_type: "arm64",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      mdm: createMockHostMdmData({ enrollment_status: enrollmentStatus }),
    });

    return render(
      <Vitals
        vitalsData={normalizeEmptyValues(mockHost)}
        mdm={mockHost.mdm}
        customHostVitals={customHostVitals}
        toggleVitalsModal={withModal ? jest.fn() : undefined}
      />
    );
  };

  const makeCustomVitals = (count: number): IHostCustomVital[] =>
    Array.from({ length: count }, (_, i) => ({
      custom_host_vital_id: i + 1,
      // "zz" so these sort after every built-in label, making them the rows the
      // cap drops; lettered rather than numbered so the sort order reads the
      // same as the suffix ("zz vital 10" would sort before "zz vital 2").
      name: `zz vital ${String.fromCharCode(97 + i)}`,
      value: `value ${i + 1}`,
    }));

  it("renders every vital an iOS host reports when they fit within two rows", () => {
    const { container } = renderCard("ios");

    const rendered = getRenderedVitals(container);

    expect(rendered).toEqual([
      "Added to Fleet",
      "Disk space available",
      "Hardware model",
      "MDM server URL",
      "MDM status",
      "Operating system",
      "Serial number",
      "Timezone",
    ]);
    expect(rendered.length).toBeLessThanOrEqual(CAP);
  });

  it("renders the same vitals for an iPadOS host", () => {
    const { container } = renderCard("ipados");

    expect(getRenderedVitals(container)).toEqual([
      "Added to Fleet",
      "Disk space available",
      "Hardware model",
      "MDM server URL",
      "MDM status",
      "Operating system",
      "Serial number",
      "Timezone",
    ]);
  });

  it("caps an iOS card at two rows once it has more vitals than fit", () => {
    const { container } = renderCard("ios", {
      customHostVitals: makeCustomVitals(10),
    });

    const rendered = getRenderedVitals(container);

    expect(rendered).toHaveLength(CAP);
    expect(isAlphabetical(rendered)).toBe(true);
    // 8 built-in vitals plus the first 4 custom ones fill the two rows; the
    // remaining 6 move behind "View all".
    expect(rendered).toContain("zz vital d");
    expect(rendered).not.toContain("zz vital e");
  });

  it("does not cap a surface without a 'View all' button, which would strand the hidden vitals", () => {
    const { container } = renderCard("ios", {
      customHostVitals: makeCustomVitals(10),
      withModal: false,
    });

    expect(getRenderedVitals(container).length).toBeGreaterThan(CAP);
    expect(
      screen.queryByRole("button", { name: "View all" })
    ).not.toBeInTheDocument();
  });

  it("does not cap a personal (BYOD) iOS host, since there's nothing extra behind 'View all'", () => {
    const { container } = renderCard("ios", {
      customHostVitals: makeCustomVitals(10),
      enrollmentStatus: "On (manual - personal)",
    });

    expect(getRenderedVitals(container).length).toBeGreaterThan(CAP);
    expect(
      screen.queryByRole("button", { name: "View all" })
    ).not.toBeInTheDocument();
  });

  it("leaves other platforms uncapped, rendering vitals outside the iOS subset", () => {
    const { container } = renderCard("darwin", {
      customHostVitals: makeCustomVitals(10),
    });

    const rendered = getRenderedVitals(container);

    // Vitals a macOS card shows that an iOS card never does.
    expect(rendered).toContain("Memory");
    expect(rendered).toContain("Processor type");
    expect(rendered).toContain("Private IP address");
    expect(rendered.length).toBeGreaterThan(CAP);
    expect(isAlphabetical(rendered)).toBe(true);
  });

  it("shows Enrollment ID instead of Serial number for a personally-enrolled iOS host", () => {
    const mockHost = createMockHost({
      platform: "ios",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual - personal)",
      }),
    });

    const { container } = render(
      <Vitals
        vitalsData={normalizeEmptyValues(mockHost)}
        mdm={mockHost.mdm}
        toggleVitalsModal={jest.fn()}
      />
    );

    const rendered = getRenderedVitals(container);

    expect(rendered).toContain("Enrollment ID");
    expect(rendered).not.toContain("Serial number");
  });
});

describe("MDM attestation", () => {
  it("renders MDM attestation when mdm_enrollment_hardware_attested is true", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      mdm_enrollment_hardware_attested: true,
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("MDM attestation")).toBeInTheDocument();
    expect(screen.getByText("Yes")).toBeInTheDocument();
  });

  it("does not render MDM attestation when mdm_enrollment_hardware_attested is false", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      mdm_enrollment_hardware_attested: false,
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.queryByText("MDM attestation")).not.toBeInTheDocument();
  });

  it("does not render MDM attestation when mdm_enrollment_hardware_attested is undefined", () => {
    const mockHost = createMockHost({
      platform: "darwin",
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.queryByText("MDM attestation")).not.toBeInTheDocument();
  });
});

describe("Disk encryption data", () => {
  it("renders 'On' for macOS when enabled", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      disk_encryption_enabled: true,
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk encryption")).toBeInTheDocument();
    expect(screen.getByText("On")).toBeInTheDocument();
  });

  it("renders 'Off' for Windows when disabled", () => {
    const mockHost = createMockHost({
      platform: "windows",
      disk_encryption_enabled: false,
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk encryption")).toBeInTheDocument();
    expect(screen.getByText("Off")).toBeInTheDocument();
  });

  it("renders 'Unknown' when disk encryption status is undefined", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      disk_encryption_enabled: undefined,
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk encryption")).toBeInTheDocument();
    expect(screen.getByText("Unknown")).toBeInTheDocument();
  });

  it("renders 'Always on' for Chrome platform", () => {
    const mockHost = createMockHost({
      platform: "chrome",
      disk_encryption_enabled: true,
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk encryption")).toBeInTheDocument();
    expect(screen.getByText("Always on")).toBeInTheDocument();
  });

  it("does not render disk encryption for unsupported platforms", () => {
    const mockHost = createMockHost({
      platform: "android",
      disk_encryption_enabled: true,
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.queryByText("Disk encryption")).not.toBeInTheDocument();
  });
});

describe("Agent data", () => {
  it("with all info present, render Agent header with orbit_version and tooltip with all 3 data points", async () => {
    const customRender = createCustomRenderer({});
    const mockHost = createMockHost({
      platform: "darwin",
      orbit_version: "1.2.0",
      osquery_version: "5.5.1",
      fleet_desktop_version: "1.0.0",
    });

    const { user } = customRender(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("1.2.0")).toBeInTheDocument();

    await user.hover(screen.getByText("1.2.0"));

    await waitFor(() => {
      expect(screen.getByText(/osquery: 5.5.1/)).toBeInTheDocument();
      expect(screen.getByText(/Orbit: 1.2.0/)).toBeInTheDocument();
      expect(screen.getByText(/Fleet Desktop: 1.0.0/)).toBeInTheDocument();
    });
  });

  it("omit fleet desktop from tooltip if no fleet desktop version", async () => {
    const customRender = createCustomRenderer({});
    const mockHost = createMockHost({
      platform: "darwin",
      orbit_version: "1.2.0",
      osquery_version: "5.5.1",
      fleet_desktop_version: DEFAULT_EMPTY_CELL_VALUE,
    });

    const { user } = customRender(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Agent")).toBeInTheDocument();

    await user.hover(screen.getByText("1.2.0"));

    await waitFor(() => {
      expect(screen.getByText(/osquery: 5.5.1/)).toBeInTheDocument();
      expect(screen.getByText(/Orbit: 1.2.0/)).toBeInTheDocument();
      expect(screen.queryByText(/Fleet desktop:/i)).not.toBeInTheDocument();
    });
  });

  it("for vanilla osquery hosts, renders Agent header with osquery_version and no tooltip", async () => {
    const osqVersion = "5.21.0";
    const customRender = createCustomRenderer({});
    const mockHost = createMockHost({
      platform: "darwin",
      orbit_version: DEFAULT_EMPTY_CELL_VALUE,
      osquery_version: osqVersion,
      fleet_desktop_version: DEFAULT_EMPTY_CELL_VALUE,
    });

    const { user } = customRender(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText(osqVersion)).toBeInTheDocument();

    await user.hover(screen.getByText(osqVersion));
    expect(screen.queryByText(/Orbit/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Fleet Desktop/i)).not.toBeInTheDocument();
  });

  it("for Chromebooks, render Agent header with osquery_version that is the fleetd chrome version and no tooltip", async () => {
    const customRender = createCustomRenderer({});
    const mockHost = createMockHost({
      platform: "chrome",
      osquery_version: "fleetd-chrome 1.2.0",
    });

    const fleetdChromeVersion = mockHost.osquery_version as string;

    const { user } = customRender(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Agent")).toBeInTheDocument();
    await user.hover(screen.getByText(new RegExp(fleetdChromeVersion, "i")));
    expect(screen.queryByText("Osquery")).not.toBeInTheDocument();
  });
});

describe("Last restarted vital", () => {
  it.each(["darwin", "windows", "ubuntu"])(
    "renders Last restarted for supported platform: %s",
    (platform) => {
      const mockHost = createMockHost({
        platform: platform as HostPlatform,
        last_restarted_at: "2023-01-01T00:00:00Z",
      });

      render(<Vitals vitalsData={mockHost} />);

      expect(screen.getByText("Last restarted")).toBeInTheDocument();
    }
  );

  it.each(["chrome", "ios", "ipados", "android"])(
    "does not render Last restarted for unsupported platform: %s",
    (platform) => {
      const mockHost = createMockHost({
        platform: platform as HostPlatform,
        last_restarted_at: "2023-01-01T00:00:00Z",
      });

      render(<Vitals vitalsData={mockHost} />);

      expect(screen.queryByText("Last restarted")).not.toBeInTheDocument();
    }
  );
});

describe("Munki version vital", () => {
  it("renders the Munki version vital when its value is a normal version string", () => {
    const mockHost = createMockHost({ platform: "darwin" });

    render(<Vitals vitalsData={mockHost} munki={{ version: "5.5.1" }} />);

    expect(screen.getByText("Munki version")).toBeInTheDocument();
    expect(screen.getByText("5.5.1")).toBeInTheDocument();
  });
});

describe("Disk space field visibility", () => {
  it("hides disk space field when storage measurement is not supported (sentinel value -1)", () => {
    const mockHost = createMockHost({
      gigs_disk_space_available: -1,
      percent_disk_space_available: 0,
      platform: "android",
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.queryByText("Disk space available")).not.toBeInTheDocument();
  });

  it("shows disk space field for zero storage (disk full)", () => {
    const mockHost = createMockHost({
      gigs_disk_space_available: 0,
      percent_disk_space_available: 0,
      platform: "android",
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk space available")).toBeInTheDocument();
  });

  it("renders disk space normally for positive values", () => {
    const mockHost = createMockHost({
      gigs_disk_space_available: 25.5,
      percent_disk_space_available: 50,
      platform: "darwin",
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.getByText("Disk space available")).toBeInTheDocument();
  });

  it("handles other negative values as not supported", () => {
    const mockHost = createMockHost({
      gigs_disk_space_available: -10,
      percent_disk_space_available: 0,
      platform: "android",
    });

    render(<Vitals vitalsData={mockHost} />);

    expect(screen.queryByText("Disk space available")).not.toBeInTheDocument();
  });
});

describe("Custom host vitals", () => {
  const customHostVitals = [
    { custom_host_vital_id: 1, name: "Asset tag", value: "FLEET-001234" },
    { custom_host_vital_id: 2, name: "Purchase date", value: "" },
  ];

  it("renders each custom host vital as a name/value row, falling back to the empty cell value when unset", () => {
    const mockHost = createMockHost({ platform: "darwin" });

    render(
      <Vitals vitalsData={mockHost} customHostVitals={customHostVitals} />
    );

    expect(screen.getByText("Asset tag")).toBeInTheDocument();
    expect(screen.getByText("FLEET-001234")).toBeInTheDocument();
    expect(screen.getByText("Purchase date")).toBeInTheDocument();
    // The vital with no value falls back to the default empty cell value.
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders values as plain text (no edit affordance) when no edit handler is provided", () => {
    const mockHost = createMockHost({ platform: "darwin" });

    render(
      <Vitals vitalsData={mockHost} customHostVitals={customHostVitals} />
    );

    expect(
      screen.queryByRole("button", { name: "Edit Asset tag" })
    ).not.toBeInTheDocument();
    expect(screen.getByText("FLEET-001234")).toBeInTheDocument();
  });

  it("renders an edit pencil next to the label and calls the edit handler on click", async () => {
    const mockHost = createMockHost({ platform: "darwin" });
    const onEditCustomHostVital = jest.fn();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={mockHost}
        customHostVitals={customHostVitals}
        onEditCustomHostVital={onEditCustomHostVital}
      />
    );

    expect(screen.getByText("FLEET-001234")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "FLEET-001234" })
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Edit Asset tag" }));

    expect(onEditCustomHostVital).toHaveBeenCalledWith(customHostVitals[0]);
  });

  describe("Operating system OS update requirement", () => {
    const renderWithRequirement = createCustomRenderer({});

    it("shows the required version and deadline", async () => {
      const mockHost = createMockHost({
        platform: "darwin",
        os_version: "macOS 26.5",
      });

      const { user } = renderWithRequirement(
        <Vitals
          vitalsData={mockHost}
          osUpdateMinimumVersion="26.6"
          osUpdateDeadline="2026-07-30"
        />
      );

      await user.hover(screen.getByText("macOS 26.5"));

      await waitFor(() => {
        const tooltip = screen.getByText(/Minimum version required:/i);
        expect(tooltip).toBeVisible();
        expect(tooltip).toHaveTextContent("Minimum version required: 26.6");
        expect(tooltip).toHaveTextContent("Deadline: 2026-07-30");
      });

      // The values are bolded, the labels aren't.
      expect(screen.getByText("26.6").tagName).toBe("B");
      expect(screen.getByText("2026-07-30").tagName).toBe("B");
    });

    it("shows Pending while the target is still being resolved", async () => {
      const mockHost = createMockHost({
        platform: "darwin",
        os_version: "macOS 26.5",
      });

      const { user } = renderWithRequirement(
        <Vitals
          vitalsData={mockHost}
          osUpdateMinimumVersion="Pending"
          osUpdateDeadline="Pending"
        />
      );

      await user.hover(screen.getByText("macOS 26.5"));

      await waitFor(() => {
        const tooltip = screen.getByText(/Minimum version required:/i);
        expect(tooltip).toBeVisible();
        expect(tooltip).toHaveTextContent("Minimum version required: Pending");
        expect(tooltip).toHaveTextContent("Deadline: Pending");
      });
    });

    it("renders no tooltip when there's no requirement", () => {
      const mockHost = createMockHost({
        platform: "darwin",
        os_version: "macOS 26.5",
      });

      renderWithRequirement(<Vitals vitalsData={mockHost} />);

      expect(screen.getByText("macOS 26.5")).toBeVisible();
      expect(screen.queryByText(/Minimum version required/i)).toBeNull();
    });
  });
});

describe("Android vitals", () => {
  const ANDROID_VITAL_TITLES = [
    "Bootloader version",
    "Encryption status",
    "Play Protect enabled",
    "Kernel version",
    "Manufacturer",
    "Passcode set",
    "Security posture",
    "Security update version",
    "Software update status",
    "USB debugging enabled",
  ];

  /** Shown only for company-owned hosts. */
  const COMPANY_OWNED_TITLES = ["Carrier", "IMEI", "Phone number"];

  /** Android hosts enroll manually; ownership is stated explicitly because the
   * identifier rows require positive confirmation of company ownership. */
  const companyOwnedMdm = () =>
    createMockHostMdmData({
      enrollment_status: "On (manual)",
      is_personal_enrollment: false,
    });

  const createMockAndroidHost = (overrides: Partial<IHost> = {}) =>
    createMockHost({
      platform: "android",
      mdm: companyOwnedMdm(),
      os_version: "Android 16 (2026-05-01)",
      adb_enabled: true,
      passcode_protected: true,
      play_protect_enabled: true,
      encryption_type: "ACTIVE",
      manufacturer: "Motorola",
      security_update_version: "2026-05-01",
      device_kernel_version: "6.1.75-android15-8-g3fb15dd41ea-ab12873456",
      bootloader_version: "S906U1UES9FYL1",
      system_update_status: "UNKNOWN_UPDATE_AVAILABLE",
      security_posture: "SECURE",
      imei: "A1000031212",
      api_level: 36,
      telephony_infos: [
        { phone_number: "18001234567", carrier_name: "Verizon" },
      ],
      ...overrides,
    });

  /** Mirrors production exactly: HostDetailsPage and DeviceUserPage both hand
   * the card normalizeEmptyValues(pick(host, HOST_VITALS_DATA)). Rendering the
   * raw host instead would hide a vital that never made it into that pick. */
  const renderAndroidCard = (
    mockHost: IHost,
    { withMdm = true }: { withMdm?: boolean } = {}
  ) =>
    render(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={withMdm ? mockHost.mdm : undefined}
      />
    );

  /** The value rendered next to a vital's title. */
  const getVitalValue = (container: HTMLElement, title: string) =>
    Array.from(
      container.querySelectorAll(".vitals-card__info-grid > .data-set > dt")
    ).find((el) => el.textContent === title)?.nextElementSibling?.textContent;

  it("renders every Android vital the device reported", () => {
    const { container } = renderAndroidCard(createMockAndroidHost());

    expect(getVitalValue(container, "Manufacturer")).toBe("Motorola");
    expect(getVitalValue(container, "Operating system")).toBe("Android 16");
    expect(getVitalValue(container, "Kernel version")).toBe(
      "6.1.75-android15-8-g3fb15dd41ea-ab12873456"
    );
    expect(getVitalValue(container, "Bootloader version")).toBe(
      "S906U1UES9FYL1"
    );
    expect(getVitalValue(container, "IMEI")).toBe("A1000031212");
    expect(getVitalValue(container, "Phone number")).toBe("18001234567");
    expect(getVitalValue(container, "Carrier")).toBe("Verizon");
    expect(getVitalValue(container, "Encryption status")).toBe("Active");
    expect(getVitalValue(container, "Security posture")).toBe("Secure");
    expect(getVitalValue(container, "Software update status")).toBe(
      "Update pending"
    );
    expect(getVitalValue(container, "Security update version")).toBe(
      "May 1, 2026"
    );
    expect(getVitalValue(container, "USB debugging enabled")).toBe("True");
    expect(getVitalValue(container, "Passcode set")).toBe("True");
    expect(getVitalValue(container, "Play Protect enabled")).toBe("True");
  });

  it("maps the AMAPI enum values to human-readable labels", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        encryption_type: "ACTIVE_PER_USER",
        security_posture: "POTENTIALLY_COMPROMISED",
        system_update_status: "SECURITY_UPDATE_AVAILABLE",
      })
    );

    expect(getVitalValue(container, "Encryption status")).toBe(
      "Active (per user)"
    );
    expect(getVitalValue(container, "Security posture")).toBe(
      "Potentially compromised"
    );
    expect(getVitalValue(container, "Software update status")).toBe(
      "Security update pending"
    );
  });

  it("falls back to the raw value for an enum member Fleet doesn't know", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        security_posture: "SOME_NEW_POSTURE" as never,
      })
    );

    expect(getVitalValue(container, "Security posture")).toBe(
      "SOME_NEW_POSTURE"
    );
  });

  it("does not resolve an enum value to an Object prototype member", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ security_posture: "constructor" as never })
    );

    expect(getVitalValue(container, "Security posture")).toBe("constructor");
  });

  it("renders booleans as True/False, including a reported false", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        adb_enabled: false,
        passcode_protected: false,
        play_protect_enabled: true,
      })
    );

    // `false` must survive the trip through normalizeEmptyValues rather than
    // collapsing into the empty-cell value.
    expect(getVitalValue(container, "USB debugging enabled")).toBe("False");
    expect(getVitalValue(container, "Passcode set")).toBe("False");
    expect(getVitalValue(container, "Play Protect enabled")).toBe("True");
  });

  it("formats the security update version as a readable date", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ security_update_version: "2026-05-01" })
    );

    // Parsed in the local timezone, so the day never rolls backwards.
    expect(getVitalValue(container, "Security update version")).toBe(
      "May 1, 2026"
    );
  });

  it("shows a security patch level that isn't date-shaped as sent", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ security_update_version: "2026-05" })
    );

    expect(getVitalValue(container, "Security update version")).toBe("2026-05");
  });

  it("does not throw on a date-shaped but impossible security patch level", () => {
    // Intl.DateTimeFormat throws on an Invalid Date, which would take down the
    // whole card rather than just this row.
    const { container } = renderAndroidCard(
      createMockAndroidHost({ security_update_version: "2026-13-45" })
    );

    expect(getVitalValue(container, "Security update version")).toBe(
      "2026-13-45"
    );
    expect(getVitalValue(container, "Manufacturer")).toBe("Motorola");
  });

  it("does not roll a nonexistent calendar date forward", () => {
    // V8 parses "2026-02-30" as March 2, so formatting it would report a patch
    // level the device never sent.
    const { container } = renderAndroidCard(
      createMockAndroidHost({ security_update_version: "2026-02-30" })
    );

    expect(getVitalValue(container, "Security update version")).toBe(
      "2026-02-30"
    );
  });

  it("renders the empty cell value for each vital the device didn't report", () => {
    const mockHost = createMockHost({
      platform: "android",
      mdm: companyOwnedMdm(),
    });

    const { container } = renderAndroidCard(mockHost);

    [...ANDROID_VITAL_TITLES, ...COMPANY_OWNED_TITLES].forEach((title) => {
      expect(getVitalValue(container, title)).toBe(DEFAULT_EMPTY_CELL_VALUE);
    });
  });

  it("does not render any Android vital for a non-Android host", () => {
    const mockHost = createMockHost({ platform: "darwin" });

    const { container } = renderAndroidCard(mockHost);

    [...ANDROID_VITAL_TITLES, ...COMPANY_OWNED_TITLES, "MEID"].forEach(
      (title) => {
        expect(getVitalValue(container, title)).toBeUndefined();
      }
    );
  });

  it("renders MEID when the device reports one", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ imei: undefined, meid: "A00000292788E1" })
    );

    expect(getVitalValue(container, "MEID")).toBe("A00000292788E1");
  });

  it("omits the MEID row on a device that doesn't report one", () => {
    const { container } = renderAndroidCard(createMockAndroidHost());

    expect(getVitalValue(container, "MEID")).toBeUndefined();
  });

  it("hides the company-owned-only vitals for a personally enrolled host", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        meid: "A00000292788E1",
        mdm: createMockHostMdmData({
          enrollment_status: "On (manual - personal)",
        }),
      })
    );

    [...COMPANY_OWNED_TITLES, "MEID"].forEach((title) => {
      expect(getVitalValue(container, title)).toBeUndefined();
    });
    // The rest of the Android vitals still render.
    expect(getVitalValue(container, "Security posture")).toBe("Secure");
  });

  it("keeps the company-owned-only vitals hidden after a personal host unenrolls", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        meid: "A00000292788E1",
        mdm: createMockHostMdmData({
          enrollment_status: "Off",
          is_personal_enrollment: true,
        }),
      })
    );

    [...COMPANY_OWNED_TITLES, "MEID"].forEach((title) => {
      expect(getVitalValue(container, title)).toBeUndefined();
    });
  });

  it("hides the company-owned-only vitals when ownership can't be determined", () => {
    // The My device page renders the card without MDM data, so BYOD can't be
    // ruled out and these hardware identifiers stay hidden.
    const { container } = renderAndroidCard(
      createMockAndroidHost({ meid: "A00000292788E1" }),
      { withMdm: false }
    );

    [...COMPANY_OWNED_TITLES, "MEID"].forEach((title) => {
      expect(getVitalValue(container, title)).toBeUndefined();
    });
    expect(getVitalValue(container, "Security posture")).toBe("Secure");
  });

  it("hides the company-owned-only vitals when the ownership flag is absent", () => {
    // is_personal_enrollment carries no omitempty server-side, so it should
    // always be present — but ownership is confirmed positively rather than
    // assumed, so a payload without it hides the identifiers.
    const mockHost = createMockAndroidHost({ meid: "A00000292788E1" });
    delete mockHost.mdm.is_personal_enrollment;

    const { container } = renderAndroidCard(mockHost);

    [...COMPANY_OWNED_TITLES, "MEID"].forEach((title) => {
      expect(getVitalValue(container, title)).toBeUndefined();
    });
    expect(getVitalValue(container, "Security posture")).toBe("Secure");
  });

  it("omits the MEID row when the server reports it as empty", () => {
    // normalizeEmptyValues rewrites an empty value to the empty-cell string,
    // which must not be mistaken for a reported MEID.
    const { container } = renderAndroidCard(
      createMockAndroidHost({ meid: "" })
    );

    expect(getVitalValue(container, "MEID")).toBeUndefined();
  });

  it("shows the Android API level in a tooltip on the operating system", async () => {
    const mockHost = createMockAndroidHost();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={mockHost.mdm}
      />
    );

    await user.hover(screen.getByText("Android 16"));

    await waitFor(() => {
      expect(screen.getByText(/Android API level: 36/)).toBeInTheDocument();
    });
  });

  it("renders no API level tooltip when the device didn't report one", async () => {
    const mockHost = createMockAndroidHost();
    // The server omits the field entirely rather than sending a null.
    delete mockHost.api_level;
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={mockHost.mdm}
      />
    );

    await user.hover(screen.getByText("Android 16"));

    await waitFor(() => {
      expect(screen.queryByText(/Android API level/)).not.toBeInTheDocument();
    });
  });

  it("lists every number in a tooltip on a dual-SIM device", async () => {
    const mockHost = createMockAndroidHost({
      telephony_infos: [
        { phone_number: "18001234567", carrier_name: "Verizon" },
        { phone_number: "18009876543", carrier_name: "AT&T" },
      ],
    });
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={mockHost.mdm}
      />
    );

    // The first SIM's number is what shows on the card.
    await user.hover(screen.getAllByText("18001234567")[0]);

    await waitFor(() => {
      expect(screen.getByText("18009876543")).toBeInTheDocument();
    });
  });

  it("shows one carrier when both SIMs are on the same network", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({
        telephony_infos: [
          { phone_number: "18001234567", carrier_name: "Verizon" },
          { phone_number: "18009876543", carrier_name: "Verizon" },
        ],
      })
    );

    // Deduplicated, so the value isn't repeated and stays usable as a key.
    expect(getVitalValue(container, "Carrier")).toBe("Verizon");
  });

  it("links out to the docs from the Security posture tooltip", async () => {
    const mockHost = createMockAndroidHost();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={mockHost.mdm}
      />
    );

    await user.hover(screen.getByText("Security posture"));

    const tooltip = await screen.findByText(
      /Google's risk assessment for this host/i
    );
    // Scoped to this tooltip so the assertion can't drift onto another vital's
    // Learn more link.
    const learnMore = within(tooltip.closest("div") as HTMLElement).getByText(
      /Learn more/
    );
    expect(learnMore.closest("a")).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/security-posture"
    );
  });

  it("capitalizes a manufacturer the device reports in lowercase", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ manufacturer: "samsung" })
    );

    expect(getVitalValue(container, "Manufacturer")).toBe("Samsung");
  });

  it("leaves a manufacturer that capitalizes itself alone", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ manufacturer: "OnePlus" })
    );

    expect(getVitalValue(container, "Manufacturer")).toBe("OnePlus");
  });

  it("drops the security patch level from the operating system version", () => {
    // The server folds it into os_version, and it has its own vital now, so
    // showing it here would just repeat it.
    const { container } = renderAndroidCard(
      createMockAndroidHost({ os_version: "Android 16 (2026-01-01)" })
    );

    expect(getVitalValue(container, "Operating system")).toBe("Android 16");
  });

  it("leaves an Android version that reports no patch level alone", () => {
    const { container } = renderAndroidCard(
      createMockAndroidHost({ os_version: "Android 16" })
    );

    expect(getVitalValue(container, "Operating system")).toBe("Android 16");
  });

  it("does not strip a parenthetical from a non-Android OS version", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      os_version: "macOS 26.5 (25F74)",
    });

    const { container } = renderAndroidCard(mockHost);

    expect(getVitalValue(container, "Operating system")).toBe(
      "macOS 26.5 (25F74)"
    );
  });

  it("explains the encryption status and software update status vitals", async () => {
    const mockHost = createMockAndroidHost();
    const customRender = createCustomRenderer({});

    const { user } = customRender(
      <Vitals
        vitalsData={normalizeEmptyValues(pick(mockHost, HOST_VITALS_DATA))}
        mdm={mockHost.mdm}
      />
    );

    await user.hover(screen.getByText("Encryption status"));
    await waitFor(() => {
      expect(
        screen.getByText(/Shows whether this device's storage is encrypted\./i)
      ).toBeInTheDocument();
    });

    await user.hover(screen.getByText("Software update status"));
    await waitFor(() => {
      expect(
        screen.getByText(
          /Shows whether an Android OS update is available for this host\./i
        )
      ).toBeInTheDocument();
    });
  });
});
