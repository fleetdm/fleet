import React from "react";
import { render, screen } from "@testing-library/react";

import AppleOSTargetForm from "./AppleOSTargetForm";

describe("AppleOSTargetForm", () => {
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
});
