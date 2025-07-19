import React from "react";
import { render, screen } from "@testing-library/react";

import createMockHost from "__mocks__/hostMock";
import { createMockHostMdmData } from "__mocks__/mdmMock";

import About from "./About";

describe("About Card component", () => {
  it("renders only the device Hardware model for Android hosts that were not enrolled in MDM personally", () => {
    const mockHost = createMockHost({
      platform: "android",
      hardware_model: "Pixel 6",
      hardware_serial: "",
    });

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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

    render(<About aboutData={mockHost} mdm={mockHost.mdm} />);

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
