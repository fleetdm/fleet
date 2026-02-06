import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockHost from "__mocks__/hostMock";
import { createMockHostMdmData } from "__mocks__/mdmMock";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Vitals from "./Vitals";

describe("Vitals Card component", () => {
  it("renders only the device Hardware model for Android hosts that were not enrolled in MDM personally", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "",
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("Pixel 6")).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
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
        enrollment_status: "On (personal)",
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

  it("renders Enrollment ID and Hardware model for personally enrolled iOS hosts", () => {
    const mockHost = createMockHost({
      platform: "ios",
      hardware_model: "iPhone 12",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("iPhone 12")).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Enrollment ID and Hardware model for personally enrolled iPad hosts", () => {
    const mockHost = createMockHost({
      platform: "ipados",
      hardware_model: "IPad Pro",
      hardware_serial: "",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("IPad Pro")).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Serial number and Hardware model for non-personally enrolled iOS hosts", () => {
    const mockHost = createMockHost({
      platform: "ios",
      hardware_model: "iPhone 12",
      hardware_serial: "123-456-789",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (manual)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("iPhone 12")).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("123-456-789")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("renders Enrollment ID and Hardware model for non-personally enrolled iPad hosts", () => {
    const mockHost = createMockHost({
      platform: "ipados",
      hardware_model: "IPad Pro",
      hardware_serial: "123-456-789",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (automatic)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("IPad Pro")).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("123-456-789")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
    expect(screen.queryByText("Private IP address")).not.toBeInTheDocument();
    expect(screen.queryByText("Public IP address")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and EnrollmentID for all non android and ios/ipad hosts that have enrolled their personal mdm devices", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBook Pro",
      hardware_serial: "",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: createMockHostMdmData({
        enrollment_status: "On (personal)",
      }),
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Enrollment ID")).toBeInTheDocument();
    expect(screen.getAllByText("enrollment-id-12345")[0]).toBeInTheDocument();
    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.queryByText("Serial number")).not.toBeInTheDocument();
  });

  it("render Hardware model, IP addresses, and Serial number for all non android and ios/ipad hosts that have enrolled not enrolled in MDM", () => {
    const mockHost = createMockHost({
      platform: "darwin",
      hardware_model: "MacBook Pro",
      hardware_serial: "test-serial-number",
      primary_ip: "192.168.1.1",
      public_ip: "203.0.113.1",
      uuid: "enrollment-id-12345",
      mdm: undefined,
    });

    render(<Vitals vitalsData={mockHost} mdm={mockHost.mdm} />);

    expect(screen.getByText("Hardware model")).toBeInTheDocument();
    expect(screen.getByText("MacBook Pro")).toBeInTheDocument();
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
      hardware_model: "MacBook Pro",
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
    expect(screen.getByText("MacBook Pro")).toBeInTheDocument();
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
      hardware_model: "MacBook Pro",
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
    expect(screen.getByText("MacBook Pro")).toBeInTheDocument();
    expect(screen.getByText("Private IP address")).toBeInTheDocument();
    expect(screen.getAllByText("192.168.1.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Public IP address")).toBeInTheDocument();
    expect(screen.getAllByText("203.0.113.1")[0]).toBeInTheDocument();
    expect(screen.getByText("Serial number")).toBeInTheDocument();
    expect(screen.getAllByText("test-serial-number")[0]).toBeInTheDocument();
    expect(screen.queryByText("Enrollment ID")).not.toBeInTheDocument();
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
