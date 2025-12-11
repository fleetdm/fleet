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
    baseUrl("/teams/1"),
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
