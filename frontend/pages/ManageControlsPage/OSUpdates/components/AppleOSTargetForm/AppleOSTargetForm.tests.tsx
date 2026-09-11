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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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

  // Every field is sent for every target: the config PATCH merges key by key,
  // so an omitted one would leave the stored value behind.
  it("sends the sentinel, no deadline and the days for 'Latest version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="7"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: /Save/i }));

    await waitFor(() => {
      expect(requestBody?.mdm?.macos_updates?.minimum_version).toBe("latest");
      expect(requestBody?.mdm?.macos_updates?.deadline).toBe("");
      expect(requestBody?.mdm?.macos_updates?.deadline_days).toBe(7);
    });
  });

  it("clears every field for 'No updates enforced'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="7"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("No updates enforced"));
    await user.click(screen.getByRole("button", { name: /Save/i }));

    await waitFor(() => {
      expect(requestBody?.mdm?.macos_updates?.minimum_version).toBe("");
      expect(requestBody?.mdm?.macos_updates?.deadline).toBe("");
      expect(requestBody?.mdm?.macos_updates?.deadline_days).toBeNull();
    });
  });

  it("nulls deadline_days when moving from 'Latest version' to 'Custom version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="7"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Custom version"));
    await user.type(screen.getByLabelText(/Minimum version/i), "15.7.8");
    await user.type(screen.getByLabelText(/^Deadline$/i), "2026-09-01");
    await user.click(screen.getByRole("button", { name: /Save/i }));

    await waitFor(() => {
      expect(requestBody?.mdm?.macos_updates?.minimum_version).toBe("15.7.8");
      expect(requestBody?.mdm?.macos_updates?.deadline).toBe("2026-09-01");
      // The stored 7 must not survive the switch out of latest mode.
      expect(requestBody?.mdm?.macos_updates?.deadline_days).toBeNull();
    });
  });

  it("sends update_new_hosts as true for 'Latest version'", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="7"
        // Stored as false, so a true in the request can only come from the
        // target rather than from the prop.
        defaultUpdateNewHosts={false}
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: /Save/i }));

    await waitFor(() => {
      expect(requestBody?.mdm?.macos_updates?.update_new_hosts).toBe(true);
    });
  });

  it("returns the checkbox to the persisted value when the target changes", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultDeadlineDays=""
        defaultUpdateNewHosts={false}
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const checkbox = screen.getByRole("checkbox", {
      name: /update_new_hosts/i,
    });

    // Tick it without saving, so the on-screen value differs from what's stored.
    await user.click(checkbox);
    await waitFor(() => expect(checkbox).toBeChecked());

    // Changing the target dismisses that unsaved input.
    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("No updates enforced"));

    expect(
      screen.getByRole("checkbox", { name: /update_new_hosts/i })
    ).not.toBeChecked();
  });

  it("seeds the days field from the stored deadline_days", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="14"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    const daysInput = screen.getByLabelText(/Days after release/i);
    expect((daysInput as HTMLInputElement).value).toBe("14");
  });

  it("keeps the 'latest' sentinel out of the minimum version input", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays="7"
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    // "latest" is the mode, not a version the user typed, so switching to a
    // custom version must start from an empty field rather than the sentinel.
    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Custom version"));

    const minVersionInput = screen.getByLabelText(/Minimum version/i);
    expect((minVersionInput as HTMLInputElement).value).toBe("");
  });

  it("renders the hardware help text when the stored version is 'latest'", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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

  it("clears a validation error when the target changes", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="latest"
        defaultDeadline=""
        defaultDeadlineDays=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: /Save/i }));
    expect(
      await screen.findByText(/The days after release is required\./i)
    ).toBeInTheDocument();

    // Away and back: the field unmounts either way, so only returning to it
    // proves the error state was cleared rather than merely hidden.
    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Custom version"));
    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Latest version"));

    expect(screen.getByLabelText(/Days after release/i)).toBeInTheDocument();
    expect(
      screen.queryByText(/The days after release is required\./i)
    ).toBeNull();
  });

  it("requires both fields for 'Custom version' rather than clearing", async () => {
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion=""
        defaultDeadline=""
        defaultDeadlineDays=""
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={jest.fn()}
      />
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(screen.getByText("Custom version"));
    await user.click(screen.getByRole("button", { name: /Save/i }));

    // Saving an empty custom form used to clear the settings, which is now what
    // "No updates enforced" is for.
    expect(
      await screen.findByText(/The minimum version is required\./i)
    ).toBeInTheDocument();
    expect(screen.getByText(/The deadline is required\./i)).toBeInTheDocument();
    expect(requestBody).toBeUndefined();
  });

  it("renders the correct form for iOS", () => {
    render(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="ios"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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
        defaultDeadlineDays=""
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

  // A rejected save must not refetch. The refetch feeds the stored config back
  // in as props and resets the form, which would discard the very input the user
  // needs to correct — e.g. a version Apple doesn't support.
  it("keeps the entered version when the server rejects the save", async () => {
    let rejected = false;
    mockServer.use(
      http.patch(baseUrl("/fleets/1"), () => {
        rejected = true;
        return HttpResponse.json(
          {
            message: "Validation Failed",
            errors: [
              {
                name: "macos_updates",
                reason: "The minimum version isn't supported by Apple.",
              },
            ],
          },
          { status: 422 }
        );
      })
    );

    const refetchTeamConfig = jest.fn();
    const { user } = renderWithBackend(
      <AppleOSTargetForm
        currentTeamId={1}
        applePlatform="darwin"
        defaultMinOsVersion="11.0"
        defaultDeadline="2024-12-31"
        defaultDeadlineDays=""
        defaultUpdateNewHosts
        refetchAppConfig={jest.fn()}
        refetchTeamConfig={refetchTeamConfig}
      />
    );

    const minVersionInput = screen.getByLabelText(/Minimum version/i);
    await user.clear(minVersionInput);
    await user.type(minVersionInput, "15.1");
    await user.click(screen.getByRole("button", { name: /Save/i }));

    await waitFor(() => {
      expect(rejected).toBe(true);
    });

    expect((minVersionInput as HTMLInputElement).value).toBe("15.1");
    expect(refetchTeamConfig).not.toHaveBeenCalled();
  });
});
