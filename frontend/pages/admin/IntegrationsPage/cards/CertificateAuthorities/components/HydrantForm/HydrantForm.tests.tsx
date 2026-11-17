import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import HydrantForm, { IHydrantFormData } from "./HydrantForm";

const createTestFormData = (overrides?: Partial<IHydrantFormData>) => ({
  name: "TEST_NAME",
  url: "https://test.com",
  clientId: "123",
  clientSecret: "test secret",
  ...overrides,
});

describe("HydrantForm", () => {
  it("render the custom button text", () => {
    render(
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeVisible();
  });

  it("enables and disables form submission depending on the form validation", async () => {
    const { user } = renderWithSetup(
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    // data is valid, so submit should be enabled
    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();

    // name input is invalidated, submit should be disabled
    await user.clear(screen.getByLabelText("Name"));
    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  const ownCA = { id: 1, name: "TEST_NAME", type: "hydrant" as const };
  const otherTypeCA = { id: 2, name: "test_name", type: "digicert" as const };
  it.each([
    [
      "adding with a name a CA of another type uses",
      [otherTypeCA],
      undefined,
      false,
    ],
    ["editing and keeping the CA's own name", [ownCA], 1, true],
    [
      "editing a CA that shares its name with another",
      [ownCA, otherTypeCA],
      1,
      false,
    ],
  ])(
    "checks the name when %s",
    (_, certAuthorities, editingCertAuthorityId, enabled) => {
      render(
        <HydrantForm
          certAuthorities={certAuthorities}
          editingCertAuthorityId={editingCertAuthorityId}
          formData={createTestFormData()}
          isSubmitting={false}
          submitBtnText="Submit"
          onChange={noop}
          onSubmit={noop}
          onCancel={noop}
        />
      );

      const submit = screen.getByRole("button", { name: "Submit" });
      if (enabled) {
        expect(submit).toBeEnabled();
      } else {
        expect(submit).toBeDisabled();
      }
    }
  );

  it("disables submit when isSubmitting is set to true", () => {
    render(
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting
        submitBtnText="Submit"
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("submit button is disabled if isDirty is false", () => {
    render(
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        isDirty={false}
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeDisabled();
  });

  it("submit button is enabled if isDirty", () => {
    render(
      <HydrantForm
        formData={createTestFormData()}
        isSubmitting={false}
        submitBtnText="Submit"
        isDirty
        onChange={noop}
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(screen.getByRole("button", { name: "Submit" })).toBeEnabled();
  });
});
