import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import IntegrationForm, { validateForm } from "./IntegrationForm";

describe("IntegrationForm validateForm", () => {
  const data = {
    url: "https://example.atlassian.net",
    apiToken: "token",
    email: "user@example.com",
    groupId: 1,
  };

  it("accepts a valid URL and email with surrounding whitespace", () => {
    const errors = validateForm(
      { ...data, url: ` ${data.url} `, email: ` ${data.email} ` },
      "zendesk"
    );

    expect(errors.url).toBeUndefined();
    expect(errors.email).toBeUndefined();
  });

  it("still rejects an invalid URL", () => {
    const errors = validateForm({ ...data, url: " not a url " }, "zendesk");

    expect(errors.url).toBe("Enter a valid HTTPS URL");
  });
});

describe("IntegrationForm validateForm for Linear", () => {
  it("skips the URL and email rules", () => {
    const errors = validateForm(
      { url: "", apiToken: "lin_api_test", teamId: "team-a" },
      "linear"
    );

    expect(errors).toEqual({});
  });

  it("requires an API key and a team ID", () => {
    const errors = validateForm(
      { url: "", apiToken: "", teamId: "" },
      "linear"
    );

    expect(errors.apiToken).toBe("Enter an API key");
    expect(errors.teamId).toBe("Enter a team ID");
  });
});

describe("IntegrationForm with the Linear destination", () => {
  const render = createCustomRenderer();
  const onSubmit = jest.fn();

  const renderLinearForm = () =>
    render(
      <IntegrationForm
        onCancel={jest.fn()}
        onSubmit={onSubmit}
        integrations={{ jira: [], zendesk: [] }}
        destination="linear"
      />
    );

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("hides the URL field and shows the API key and team ID fields", () => {
    renderLinearForm();

    expect(screen.queryByLabelText("URL")).not.toBeInTheDocument();
    expect(screen.getByLabelText("API key")).toBeInTheDocument();
    expect(screen.getByLabelText("Team ID")).toBeInTheDocument();
  });

  it("shows errors when the API key and team ID are empty", async () => {
    renderLinearForm();

    await userEvent.click(screen.getByRole("button", { name: "Add" }));

    expect(await screen.findByText("Enter an API key")).toBeInTheDocument();
    expect(screen.getByText("Enter a team ID")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with the Linear integration and destination", async () => {
    renderLinearForm();

    await userEvent.type(screen.getByLabelText("API key"), "lin_api_test");
    await userEvent.type(screen.getByLabelText("Team ID"), "team-a");
    await userEvent.click(screen.getByRole("button", { name: "Add" }));

    expect(onSubmit).toHaveBeenCalledWith(
      [{ api_key: "lin_api_test", team_id: "team-a" }],
      "linear"
    );
  });
});
