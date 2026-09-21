import React from "react";
import { createCustomRenderer, renderWithSetup } from "test/test-utils";
import { screen, render } from "@testing-library/react";
import { noop } from "lodash";

import InputField from "components/forms/fields/InputField";

import LabelForm from "./LabelForm";

const renderInGitOpsMode = createCustomRenderer({
  context: {
    app: {
      config: {
        gitops: {
          gitops_mode_enabled: true,
          repository_url: "https://github.com/example/fleet-gitops",
        },
      },
    },
  },
});

describe("LabelForm", () => {
  it("should validate the name to be required", async () => {
    const { user } = renderWithSetup(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        immutableFields={[]}
        teamName={null}
      />
    );

    const nameInput = screen.getByLabelText("Name");

    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByText("Label name must be present")).toBeInTheDocument();

    await user.type(nameInput, "Label name");
    expect(
      screen.queryByText("Label name must be present")
    ).not.toBeInTheDocument();
  });

  it("should render any additional field the user provides", () => {
    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        teamName={null}
        immutableFields={[]}
        additionalFields={
          <InputField name="test field" label="test field" value="" />
        }
      />
    );

    expect(screen.getByLabelText("test field")).toBeInTheDocument();
  });

  it("should pass up the form data when the form is submitted and valid", async () => {
    const onSave = jest.fn();
    const { user } = renderWithSetup(
      <LabelForm
        onSave={onSave}
        onCancel={jest.fn()}
        teamName={null}
        immutableFields={[]}
      />
    );

    const nameValue = "Test Name";
    const descriptionValue = "Test Description";
    await user.type(screen.getByLabelText("Name"), nameValue);
    await user.type(screen.getByLabelText("Description"), descriptionValue);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith(
      { name: nameValue, description: descriptionValue },
      true
    );
  });

  it("should not render immutable help text when no immutable fields are provided (ManualLabelForm without fleet)", () => {
    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        teamName={null}
        immutableFields={[]}
      />
    );

    // Help text container should not be in the document
    expect(
      screen.queryByText(
        /are immutable\. To make changes, delete this label and create a new one\./
      )
    ).not.toBeInTheDocument();
  });

  it("should render correct immutable help text for a single field (ManualLabelForm with fleet)", () => {
    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        teamName={"Example Fleet"}
        immutableFields={["fleets"]}
      />
    );

    expect(
      screen.getByText(
        "Label fleets are immutable. To make changes, delete this label and create a new one."
      )
    ).toBeInTheDocument();
  });

  it("should render correct immutable help text for two fields (DynamicLabelForm without fleet)", () => {
    const immutableFields = ["queries", "platforms"];

    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        teamName={null}
        immutableFields={immutableFields}
      />
    );

    expect(
      screen.getByText(
        "Label queries and platforms are immutable. To make changes, delete this label and create a new one."
      )
    ).toBeInTheDocument();

    expect(immutableFields.length).toBe(2);
  });

  it("should render correct immutable help text for three fields (DynamicLabelForm with fleet)", () => {
    render(
      <LabelForm
        onSave={noop}
        onCancel={noop}
        teamName={"Example Fleet"}
        immutableFields={["fleets", "queries", "platforms"]}
      />
    );

    expect(
      screen.getByText(
        "Label fleets, queries, and platforms are immutable. To make changes, delete this label and create a new one."
      )
    ).toBeInTheDocument();
  });

  describe("GitOps mode", () => {
    // Default path: dynamic and host vitals labels, plus label creation. Save only unlocks when
    // ManualLabelForm passes gitOpsLocksDefinitionOnly on the edit page.
    it("should keep Save gated when gitOpsLocksDefinitionOnly is not set", () => {
      renderInGitOpsMode(
        <LabelForm
          onSave={noop}
          onCancel={noop}
          teamName={null}
          immutableFields={[]}
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
      expect(screen.getByLabelText("Name")).toBeEnabled();
    });

    it("should lock only the definition fields when git owns the definition alone", () => {
      renderInGitOpsMode(
        <LabelForm
          onSave={noop}
          onCancel={noop}
          teamName={null}
          immutableFields={[]}
          gitOpsLocksDefinitionOnly
        />
      );

      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
      expect(screen.getByLabelText("Name")).toBeDisabled();
      expect(screen.getByLabelText("Description")).toBeDisabled();
    });
  });
});
