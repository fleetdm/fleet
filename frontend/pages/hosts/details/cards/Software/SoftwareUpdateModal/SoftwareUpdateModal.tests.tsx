import React from "react";
import { render, screen } from "@testing-library/react";

import { noop } from "lodash";
import { createMockHostSoftware } from "__mocks__/hostMock";
import SoftwareUpdatesModal from "./SoftwareUpdateModal";

describe("SoftwareUpdatesModal", () => {
  it("TODO", () => {
    const mockSoftware = createMockHostSoftware();
    render(
      <SoftwareUpdatesModal
        hostDisplayName="Test Host"
        software={mockSoftware}
        onExit={noop}
        onUpdate={noop}
      />
    );

    // Modal title
    expect(screen.getByText("Update details")).toBeVisible();

    // Update and cancel button
    expect(screen.getByRole("button", { name: "Update" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeVisible();
  });
});
