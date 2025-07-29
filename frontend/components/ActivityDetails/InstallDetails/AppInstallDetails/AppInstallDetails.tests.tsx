import { render, screen } from "@testing-library/react";
import { getStatusMessage } from "./AppInstallDetails";

describe("getStatusMessage helper function", () => {
  it("shows NotNow message when isStatusNotNow is true", () => {
    render(
      getStatusMessage({
        displayStatus: "pending",
        isMDMStatusNotNow: true,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
      })
    );
    expect(screen.getByText(/Fleet tried to install/i)).toBeInTheDocument();
    expect(screen.getByText(/Marko's MacBook Pro/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /but couldn't because the host was locked or was running on battery power while in Power Nap/i
      )
    ).toBeInTheDocument();
  });

  it("shows pending acknowledged message", () => {
    render(
      getStatusMessage({
        displayStatus: "pending_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
      })
    );
    expect(
      screen.getByText(/The MDM command \(request\) to install/i)
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /was acknowledged but the installation has not been verified/i
      )
    ).toBeInTheDocument();
    expect(screen.getByText(/Refetch/i)).toBeInTheDocument();
  });

  it("shows failed_install message", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
      })
    );
    expect(
      screen.getByText(/failed. Please re-attempt this installation/i)
    ).toBeInTheDocument();
  });

  it("shows failed verification message", () => {
    render(
      getStatusMessage({
        displayStatus: "failed_install",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
      })
    );
    expect(
      screen.getByText(
        /but the installation has not been verified. Please re-attempt this installation/i
      )
    ).toBeInTheDocument();
  });

  it("shows default message for installed status", () => {
    render(
      getStatusMessage({
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: true,
        appName: "Logic Pro",
        hostDisplayName: "Marko's MacBook Pro",
      })
    );
    expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
    expect(screen.getByText(/Logic Pro/i)).toBeInTheDocument();
    expect(screen.getByText(/Marko's MacBook Pro/i)).toBeInTheDocument();
  });

  it("shows default message with 'the host' if host_display_name is empty", () => {
    render(
      getStatusMessage({
        displayStatus: "installed",
        isMDMStatusNotNow: false,
        isMDMStatusAcknowledged: false,
        appName: "Logic Pro",
        hostDisplayName: "",
      })
    );
    expect(screen.getByText(/Fleet installed/i)).toBeInTheDocument();
    expect(screen.getByText(/the host/i)).toBeInTheDocument();
  });
});
