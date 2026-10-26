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

    expect(
      screen.getByText(/always available in self-service \(Google Play Store\)/)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /setup experience/ })
    ).toHaveAttribute(
      "href",
      expect.stringContaining("/controls/setup-experience/install-software")
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

  it("keeps Add software enabled and shows inline errors on submit instead of submitting", async () => {
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
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(submitButton).toBeEnabled();

    await user.click(submitButton);

    expect(onSubmit).not.toHaveBeenCalled();
    expect(screen.getByText("Enter an application ID")).toBeInTheDocument();
    expect(screen.getByText("Select at least one label")).toBeInTheDocument();
  });

  it("clears the label error once a label is selected", async () => {
    const render = createCustomRenderer();
    const { user } = render(
      <SoftwareAndroidForm
        labels={labels}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
      />
    );
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    await user.click(screen.getByRole("button", { name: "Add software" }));
    expect(screen.getByText("Select at least one label")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Android pilot" }));

    expect(
      screen.queryByText("Select at least one label")
    ).not.toBeInTheDocument();
  });

  it.each([
    ["Include any", "labelsIncludeAny"],
    ["Include all", "labelsIncludeAll"],
    ["Exclude any", "labelsExcludeAny"],
  ])(
    "submits a %s target with its labels",
    async (optionLabel, customTarget) => {
      const onSubmit = jest.fn();
      const render = createCustomRenderer();
      const { user } = render(
        <SoftwareAndroidForm
          labels={labels}
          onSubmit={onSubmit}
          onCancel={jest.fn()}
        />
      );

      await user.type(
        screen.getByRole("textbox", { name: "Application ID" }),
        " com.google.android.gm "
      );
      await user.click(screen.getByRole("radio", { name: "Custom" }));
      if (optionLabel !== "Include any") {
        await user.click(screen.getByRole("option", { name: "Include any" }));
        await user.click(screen.getByRole("option", { name: optionLabel }));
      }
      await user.click(screen.getByRole("checkbox", { name: "Android pilot" }));
      await user.click(screen.getByRole("button", { name: "Add software" }));

      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          applicationID: "com.google.android.gm",
          targetType: "Custom",
          customTarget,
          labelTargets: { "Android pilot": true },
        })
      );
    }
  );

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
