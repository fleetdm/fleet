import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import configAPI from "services/entities/config";
import { createCustomRenderer } from "test/test-utils";

import GoogleWorkspaceSection from "./GoogleWorkspaceSection";

// Premium gating now lives in the parent IdentityProviders component, so this
// section always renders its form.
const renderWith = () =>
  createCustomRenderer({
    withBackendMock: true,
    context: { app: { config: createMockConfig() } },
  });

const configuredConfig = (
  additionalDomains?: string[],
  domain = "example.com"
): IConfig => {
  const config = createMockConfig();
  // createMockConfig copies only the top level, so copy integrations before
  // changing it to keep the shared default mock unconfigured.
  config.integrations = {
    ...config.integrations,
    google_workspace: [
      {
        domain,
        additional_domains: additionalDomains,
        impersonated_user_email: "admin@example.com",
        api_key_json: { client_email: "********", private_key: "********" },
      },
    ],
  };
  return config;
};

const expectedUpdate = (additionalDomains: string[]) => ({
  integrations: {
    google_workspace: [
      {
        domain: "example.com",
        additional_domains: additionalDomains,
        impersonated_user_email: "admin@example.com",
      },
    ],
  },
});

describe("GoogleWorkspaceSection", () => {
  let updateSpy: jest.SpyInstance;
  beforeEach(() => {
    updateSpy = jest
      .spyOn(configAPI, "update")
      .mockResolvedValue(createMockConfig());
  });
  afterEach(() => {
    updateSpy.mockRestore();
  });

  it("renders the connect form without the domain list when not configured", () => {
    const render = renderWith();
    render(<GoogleWorkspaceSection appConfig={createMockConfig()} />);

    expect(screen.getByText("Google Workspace")).toBeInTheDocument();
    expect(screen.getByLabelText("API key JSON")).toBeInTheDocument();
    expect(screen.getByLabelText("Primary domain")).toBeInTheDocument();
    expect(
      screen.getByLabelText("Admin email to impersonate")
    ).toBeInTheDocument();
    // Mutual-exclusion messaging.
    expect(screen.getByText(/SCIM provisioning/i)).toBeInTheDocument();
    expect(screen.queryByText("Additional domains")).not.toBeInTheDocument();
  });

  it("pre-fills the form and masks the API key from existing config", () => {
    const render = renderWith();
    render(<GoogleWorkspaceSection appConfig={configuredConfig()} />);

    expect(screen.getByDisplayValue("example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("admin@example.com")).toBeInTheDocument();
    // Masked key shown as the unchanged-password placeholder.
    expect(screen.getByDisplayValue("********")).toBeInTheDocument();
  });

  it("shows an empty state when there are no additional domains", () => {
    const render = renderWith();
    render(<GoogleWorkspaceSection appConfig={configuredConfig()} />);

    expect(screen.getByText("No additional domains")).toBeInTheDocument();
  });

  it("lists additional domains", () => {
    const render = renderWith();
    render(
      <GoogleWorkspaceSection
        appConfig={configuredConfig(["example.org", "example.net"])}
      />
    );

    expect(screen.getByText("Additional domains")).toBeInTheDocument();
    expect(screen.getByText("example.org")).toBeInTheDocument();
    expect(screen.getByText("example.net")).toBeInTheDocument();
  });

  it("keeps saved additional domains when the form is saved", async () => {
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith(expectedUpdate(["example.org"]))
    );
  });

  it("adds a domain from the modal", async () => {
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    await user.click(screen.getByRole("button", { name: /Add/ }));
    await user.type(screen.getByLabelText("Domain"), " example.net ");
    const addButtons = screen.getAllByRole("button", { name: /Add/ });
    await user.click(addButtons[addButtons.length - 1]);

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith(
        expectedUpdate(["example.org", "example.net"])
      )
    );
  });

  it("requires a domain on submit", async () => {
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    await user.click(screen.getByRole("button", { name: /Add/ }));
    const addButtons = screen.getAllByRole("button", { name: /Add/ });
    const submit = addButtons[addButtons.length - 1];
    expect(submit).toBeEnabled();
    await user.click(submit);

    expect(screen.getByText("Enter a domain")).toBeInTheDocument();
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it("rejects a domain that's already added on blur and clears the error on focus", async () => {
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    await user.click(screen.getByRole("button", { name: /Add/ }));
    const input = screen.getByLabelText("Domain");
    await user.type(input, "EXAMPLE.com");
    // No error while typing.
    expect(
      screen.queryByText("Enter a domain that hasn't been added")
    ).not.toBeInTheDocument();

    await user.tab();
    expect(
      screen.getByText("Enter a domain that hasn't been added")
    ).toBeInTheDocument();

    await user.click(screen.getByDisplayValue("EXAMPLE.com"));
    expect(
      screen.queryByText("Enter a domain that hasn't been added")
    ).not.toBeInTheDocument();

    const addButtons = screen.getAllByRole("button", { name: /Add/ });
    await user.click(addButtons[addButtons.length - 1]);
    expect(
      screen.getByText("Enter a domain that hasn't been added")
    ).toBeInTheDocument();
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it("keeps unsaved form edits when only the additional domains change", async () => {
    const render = renderWith();
    const { user, rerender } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    const email = screen.getByLabelText("Admin email to impersonate");
    await user.clear(email);
    await user.type(email, "other@example.com");

    rerender(
      <GoogleWorkspaceSection
        appConfig={configuredConfig(["example.org", "example.net"])}
      />
    );
    expect(screen.getByDisplayValue("other@example.com")).toBeInTheDocument();

    // A change to a saved form field still resets the form.
    rerender(
      <GoogleWorkspaceSection
        appConfig={configuredConfig(["example.org"], "example.io")}
      />
    );
    expect(screen.getByDisplayValue("example.io")).toBeInTheDocument();
    expect(screen.getByDisplayValue("admin@example.com")).toBeInTheDocument();
  });

  it("disables adding and deleting domains while Save is in flight", async () => {
    updateSpy.mockReturnValue(new Promise(() => undefined));
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection appConfig={configuredConfig(["example.org"])} />
    );

    expect(screen.getByRole("button", { name: /Add/ })).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /Add/ })).toBeDisabled()
    );
    expect(
      screen.getByRole("button", { name: "Delete domain example.org" })
    ).toBeDisabled();
  });

  it("deletes a domain from the modal", async () => {
    const render = renderWith();
    const { user } = render(
      <GoogleWorkspaceSection
        appConfig={configuredConfig(["example.org", "example.net"])}
      />
    );

    await user.click(
      screen.getByRole("button", { name: "Delete domain example.org" })
    );
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith(expectedUpdate(["example.net"]))
    );
  });
});
