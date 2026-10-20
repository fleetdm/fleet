import { screen } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { ILabelSummary } from "interfaces/label";
import { createCustomRenderer } from "test/test-utils";

import SoftwareAndroidForm from "./SoftwareAndroidForm";

const labels: ILabelSummary[] = [
  { id: 1, name: "Android pilot", label_type: "regular" },
  { id: 2, name: "Department: Engineering", label_type: "regular" },
];

describe("SoftwareAndroidForm", () => {
  it("defaults the target to All hosts and hides the label list", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareAndroidForm
        labels={labels}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByText("Target")).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "All hosts" })).toBeChecked();
    expect(
      screen.queryByRole("checkbox", { name: "Android pilot" })
    ).not.toBeInTheDocument();
  });

  it("shows the custom target options and labels when Custom is selected", async () => {
    const render = createCustomRenderer();
    const { user } = render(
      <SoftwareAndroidForm
        labels={labels}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
      />
    );

    await user.click(screen.getByRole("radio", { name: "Custom" }));

    expect(screen.getByRole("option", { name: "Include any" })).toBeVisible();
    expect(
      screen.getByRole("checkbox", { name: "Android pilot" })
    ).toBeVisible();
    expect(
      screen.getByRole("checkbox", { name: "Department: Engineering" })
    ).toBeVisible();
  });

  it("disables submit for a custom target until a label is selected, then submits the label target", async () => {
    const onSubmit = jest.fn();
    const render = createCustomRenderer();
    const { user } = render(
      <SoftwareAndroidForm
        labels={labels}
        onSubmit={onSubmit}
        onCancel={jest.fn()}
      />
    );
    const submitButton = screen.getByRole("button", { name: "Add software" });

    await user.type(
      screen.getByRole("textbox", { name: "Application ID" }),
      "com.google.android.gm"
    );
    expect(submitButton).toBeEnabled();

    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(submitButton).toBeDisabled();

    await user.click(screen.getByRole("checkbox", { name: "Android pilot" }));
    expect(submitButton).toBeEnabled();

    await user.click(submitButton);

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        applicationID: "com.google.android.gm",
        targetType: "Custom",
        customTarget: "labelsIncludeAny",
        labelTargets: { "Android pilot": true },
      })
    );
  });

  it("disables the target options in GitOps mode", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "https://example.com/repo",
              exceptions: { labels: false, software: false, secrets: false },
            },
          }),
        },
      },
    });
    render(
      <SoftwareAndroidForm
        labels={labels}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
      />
    );

    expect(screen.getByRole("radio", { name: "All hosts" })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Custom" })).toBeDisabled();
  });
});
