import React from "react";
import { render, screen, waitFor } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import AppleOSTargetForm from "./AppleOSTargetForm";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

describe("AppleOSTargetForm", () => {
  let requestBody: any;
  const renderWithBackend = createCustomRenderer({
    withBackendMock: true,
  });
  const updateTeamConfigHandler = http.patch(
    baseUrl("/fleets/1"),
    async ({ request }) => {
      requestBody = await request.json();
      return HttpResponse.json({});
    }
  );

  beforeEach(() => {
    requestBody = undefined;
    mockServer.use(updateTeamConfigHandler);
  });

  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders the correct form for MacOS", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const minVersionInput = screen.getByLabelText(/Minimum version/i);
    expect(minVersionInput).toBeInTheDocument();
    expect((minVersionInput as HTMLInputElement).value).toBe("11.0");

    const deadlineInput = screen.getByLabelText(/Deadline/i);
    expect(deadlineInput).toBeInTheDocument();
    expect((deadlineInput as HTMLInputElement).value).toBe("2024-12-31");

    const updateNewHostsCheckbox = screen.getByLabelText(
      /Update new hosts to latest/i
    );
    expect(updateNewHostsCheckbox).toBeInTheDocument();
    expect((updateNewHostsCheckbox as HTMLInputElement).checked).toBe(true);
  });

  it("saves 'update new hosts' checkbox state correctly for macOS", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );
    const saveButton = screen.getByRole("button", { name: /Save/i });
    expect(saveButton).toBeInTheDocument();
    await user.click(saveButton);
    await waitFor(() => {
      expect(requestBody).toBeDefined();
      expect(requestBody?.mdm?.macos_updates?.update_new_hosts).toBe(true);
      expect(requestBody?.mdm?.macos_updates?.minimum_version).toBe("11.0");
      expect(requestBody?.mdm?.macos_updates?.deadline).toBe("2024-12-31");
    });

    const updateNewHostsCheckbox = screen.getByRole("checkbox", {
      name: /update_new_hosts/i,
    });
    await user.click(updateNewHostsCheckbox);
    await waitFor(() => {
      expect(updateNewHostsCheckbox).not.toBeChecked();
    });
    await user.click(saveButton);
    await waitFor(() => {
      expect(requestBody).toBeDefined();
      expect(requestBody?.mdm?.macos_updates?.update_new_hosts).toBe(false);
      expect(requestBody?.mdm?.macos_updates?.minimum_version).toBe("11.0");
      expect(requestBody?.mdm?.macos_updates?.deadline).toBe("2024-12-31");
    });
  });

  it("renders the hardware help text when the stored version is 'latest'", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.getByText(/Based on host hardware\./i)).toBeVisible();
    expect(screen.getByRole("link", { name: /Learn more/i })).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/apple-available-os-updates"
    );
  });

  it("shows the hardware help text only once 'Latest version' is selected", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.queryByText(/Based on host hardware\./i)).toBeNull();

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Latest version"));

    expect(screen.getByText(/Based on host hardware\./i)).toBeVisible();
  });

  it("does not render the hardware help text when no updates are enforced", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion=""
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.queryByText(/Based on host hardware\./i)).toBeNull();
  });

  it("hides the hardware help text when switching away from 'Latest version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.getByText(/Based on host hardware\./i)).toBeVisible();

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Custom version"));
    expect(screen.queryByText(/Based on host hardware\./i)).toBeNull();

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("No updates enforced"));
    expect(screen.queryByText(/Based on host hardware\./i)).toBeNull();
  });

  it("shows only the days field when 'Latest version' is selected", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.getByLabelText(/Days after release/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/Minimum version/i)).toBeNull();
    expect(screen.queryByLabelText(/^Deadline$/i)).toBeNull();
  });

  it("shows no version fields when no updates are enforced", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion=""
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.queryByLabelText(/Minimum version/i)).toBeNull();
    expect(screen.queryByLabelText(/^Deadline$/i)).toBeNull();
    expect(screen.queryByLabelText(/Days after release/i)).toBeNull();
  });

  it("swaps the fields when the target changes", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    expect(screen.getByLabelText(/Minimum version/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/Days after release/i)).toBeNull();

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Latest version"));

    expect(screen.getByLabelText(/Days after release/i)).toBeInTheDocument();
    expect(screen.queryByLabelText(/Minimum version/i)).toBeNull();
  });

  it("checks and disables 'update new hosts' for 'Latest version', leaving it visible", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultUpdateNewHosts={false}
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    // The native input is hidden by the Checkbox component's styling, so the
    // label is what proves the control is still on screen.
    expect(screen.getByText(/Update new hosts to latest/i)).toBeVisible();

    const checkbox = screen.getByLabelText(/Update new hosts to latest/i);
    expect(checkbox).toBeChecked();
    expect(checkbox).toBeDisabled();
  });

  it("leaves 'update new hosts' editable for 'Custom version'", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultUpdateNewHosts={false}
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const checkbox = screen.getByLabelText(/Update new hosts to latest/i);
    expect(checkbox).not.toBeChecked();
    expect(checkbox).toBeEnabled();
  });

  it("shows the ADE tooltip for 'Latest version' on the checkbox", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.hover(screen.getByText(/Update new hosts to latest/i));
    await waitFor(() => {
      expect(
        screen.getByText(/all hosts will be updated to latest macOS version\./i)
      ).toBeInTheDocument();
    });
  });

  it("shows the minimum version tooltip on the checkbox for 'Custom version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.hover(screen.getByText(/Update new hosts to latest/i));
    await waitFor(() => {
      expect(
        screen.getByText(/hosts below the minimum version are updated/i)
      ).toBeInTheDocument();
    });
  });

  it("shows the days after release tooltip", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.hover(screen.getByText(/Days after release/i));
    await waitFor(() => {
      expect(
        screen.getByText(
          /number of days after Apple releases an update before hosts are required to install it\./i
        )
      ).toBeInTheDocument();
    });
  });

  it("requires a value in days after release before saving", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: /Save/i }));

    expect(
      await screen.findByText(/The days after release is required\./i)
    ).toBeInTheDocument();
    expect(requestBody).toBeUndefined();
  });

  it("rejects a days after release value below 1", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.type(screen.getByLabelText(/Days after release/i), "0");
    await user.click(screen.getByRole("button", { name: /Save/i }));

    expect(
      await screen.findByText(/must be a whole number of 1 or more\./i)
    ).toBeInTheDocument();
    expect(requestBody).toBeUndefined();
  });

  it("rejects a fractional days after release value", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.type(screen.getByLabelText(/Days after release/i), "1.5");
    await user.click(screen.getByRole("button", { name: /Save/i }));

    expect(
      await screen.findByText(/must be a whole number of 1 or more\./i)
    ).toBeInTheDocument();
    expect(requestBody).toBeUndefined();
  });

  it("does not validate the hidden version fields in 'Latest version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.type(screen.getByLabelText(/Days after release/i), "7");
    await user.click(screen.getByRole("button", { name: /Save/i }));

    // The minimum version and deadline are empty but not on screen, so they
    // must not block the save.
    expect(screen.queryByText(/The minimum version is required\./i)).toBeNull();
    expect(screen.queryByText(/The deadline is required\./i)).toBeNull();
    await waitFor(() => expect(requestBody).toBeDefined());
  });

  it("renders the correct form for iOS", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="ios"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const minVersionInput = screen.getByLabelText(/Minimum version/i);
    expect(minVersionInput).toBeInTheDocument();
    expect((minVersionInput as HTMLInputElement).value).toBe("11.0");

    const deadlineInput = screen.getByLabelText(/Deadline/i);
    expect(deadlineInput).toBeInTheDocument();
    expect((deadlineInput as HTMLInputElement).value).toBe("2024-12-31");

    const updateNewHostsCheckbox = screen.queryByLabelText(
      /Update new hosts to latest/i
    );
    expect(updateNewHostsCheckbox).not.toBeInTheDocument();
  });

  it("saves 'update new hosts' checkbox state correctly for iOS", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="ios"
        defaultMinOsVersion="12.0"
        defaultDeadline="2025-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );
    const saveButton = screen.getByRole("button", { name: /Save/i });
    expect(saveButton).toBeInTheDocument();
    await user.click(saveButton);
    await waitFor(() => {
      expect(requestBody).toBeDefined();
      expect(requestBody?.mdm?.ios_updates?.update_new_hosts).not.toBeDefined();
      expect(requestBody?.mdm?.ios_updates?.minimum_version).toBe("12.0");
      expect(requestBody?.mdm?.ios_updates?.deadline).toBe("2025-12-31");
    });
  });

  it("renders the correct form for iPadOS", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="ipados"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const minVersionInput = screen.getByLabelText(/Minimum version/i);
    expect(minVersionInput).toBeInTheDocument();
    expect((minVersionInput as HTMLInputElement).value).toBe("11.0");

    const deadlineInput = screen.getByLabelText(/Deadline/i);
    expect(deadlineInput).toBeInTheDocument();
    expect((deadlineInput as HTMLInputElement).value).toBe("2024-12-31");

    const updateNewHostsCheckbox = screen.queryByLabelText(
      /Update new hosts to latest/i
    );
    expect(updateNewHostsCheckbox).not.toBeInTheDocument();
  });

  it("saves 'update new hosts' checkbox state correctly for iPadOS", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="ipados"
        defaultMinOsVersion="13.0"
        defaultDeadline="2026-12-31"
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );
    const saveButton = screen.getByRole("button", { name: /Save/i });
    expect(saveButton).toBeInTheDocument();
    await user.click(saveButton);
    await waitFor(() => {
      expect(requestBody).toBeDefined();
      expect(
        requestBody?.mdm?.ipados_updates?.update_new_hosts
      ).not.toBeDefined();
      expect(requestBody?.mdm?.ipados_updates?.minimum_version).toBe("13.0");
      expect(requestBody?.mdm?.ipados_updates?.deadline).toBe("2026-12-31");
    });
  });
});
