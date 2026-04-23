import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import DataCollection from "./DataCollection";

describe("DataCollection", () => {
  const mockHandleSubmit = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders both checkboxes with values from appConfig", () => {
    const mockConfig = createMockConfig({
      features: {
        enable_host_users: true,
        enable_software_inventory: true,
        enable_conditional_access: true,
        enable_conditional_access_bypass: true,
        data_collection: { uptime: false, cve: true },
      },
    });

    renderWithSetup(
      <DataCollection
        appConfig={mockConfig}
        handleSubmit={mockHandleSubmit}
        isPremiumTier
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

    expect(screen.getByText("Data collection")).toBeInTheDocument();
    expect(screen.getByLabelText(/hosts active/i)).not.toBeChecked();
    expect(screen.getByLabelText(/vulnerabilities/i)).toBeChecked();
  });

  it("calls handleSubmit with the flipped values", async () => {
    const mockConfig = createMockConfig({
      features: {
        enable_host_users: true,
        enable_software_inventory: true,
        enable_conditional_access: true,
        enable_conditional_access_bypass: true,
        data_collection: { uptime: true, cve: true },
      },
    });

    const { user } = renderWithSetup(
      <DataCollection
        appConfig={mockConfig}
        handleSubmit={mockHandleSubmit}
        isPremiumTier
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

    await user.click(screen.getByLabelText(/vulnerabilities/i));
    await user.click(screen.getByRole("button", { name: /save/i }));

    expect(mockHandleSubmit).toHaveBeenCalledWith({
      features: {
        data_collection: { uptime: true, cve: false },
      },
    });
  });

  it("disables the save button when gitops mode is enabled", () => {
    const mockConfig = createMockConfig({
      gitops: {
        gitops_mode_enabled: true,
        repository_url: "",
        exceptions: { labels: false, software: false, secrets: true },
      },
    });

    renderWithSetup(
      <DataCollection
        appConfig={mockConfig}
        handleSubmit={mockHandleSubmit}
        isPremiumTier
        isUpdatingSettings={false}
        router={createMockRouter()}
      />
    );

    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });
});
