import { render, screen } from "@testing-library/react";
import React from "react";

import createMockHost from "__mocks__/hostMock";

import AutoEnrollMdmModal from "./AutoEnrollMdmModal";

describe("AutoEnrollMdmModal", () => {
  const renderModal = (osName: string, osVersion: string) =>
    render(
      <AutoEnrollMdmModal
        host={createMockHost({
          platform: "darwin",
          os_name: osName,
          os_version: osVersion,
        })}
        onCancel={jest.fn()}
      />
    );

  it.each([
    ["26.1", "Enroll in Device Management"],
    ["14.6.1", "Enroll in Remote Management"],
  ])("shows the System Settings steps for macOS %s", (osVersion, enrollCTA) => {
    renderModal("macOS", osVersion);

    expect(
      screen.getByText(/From the Apple menu in the top left corner/)
    ).toBeInTheDocument();
    expect(screen.getAllByText(enrollCTA)).toHaveLength(2);
  });

  it("shows the notification center steps before macOS 14", () => {
    renderModal("macOS", "13.6.2");

    expect(
      screen.getByText(/Open your Mac's notification center/)
    ).toBeInTheDocument();
  });
});
