import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";
import { InjectedRouter } from "react-router";

import { createMockConfig } from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import Sso from "./Sso";

const createTestRenderer = () =>
  createCustomRenderer({
    context: { app: { isPremiumTier: true } },
  });

const renderSso = (subsection = "fleet-users") => {
  const router = ({ push: jest.fn() } as unknown) as InjectedRouter;
  const render = createTestRenderer();
  const { user } = render(
    <Sso
      appConfig={createMockConfig()}
      handleSubmit={jest.fn().mockResolvedValue(true)}
      isPremiumTier
      isUpdatingSettings={false}
      router={router}
      subsection={subsection}
    />
  );

  return { user, router };
};

const confirmSpy = jest.spyOn(window, "confirm");

beforeEach(() => {
  confirmSpy.mockClear();
  confirmSpy.mockImplementation(() => true);
});

afterAll(() => {
  confirmSpy.mockRestore();
});

describe("Sso - Fleet users", () => {
  it("does not prompt on tab switch when a checkbox is toggled back to its original state", async () => {
    const { user, router } = renderSso();

    const enableSso = screen.getByRole("checkbox", { name: "enableSso" });
    await user.click(enableSso);
    await user.click(enableSso);

    await user.click(screen.getByText("End users"));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(router.push).toHaveBeenCalled();
  });

  it("prompts on tab switch when the form has unsaved changes", async () => {
    const { user } = renderSso();

    await user.click(screen.getByRole("checkbox", { name: "enableSso" }));
    await user.click(screen.getByText("End users"));

    expect(confirmSpy).toHaveBeenCalled();
  });

  it("does not prompt on tab switch when a text field is edited back to its original value", async () => {
    const { user } = renderSso();

    const idpName = screen.getByLabelText("Identity provider name");
    await user.type(idpName, "Okta");
    await user.clear(idpName);

    await user.click(screen.getByText("End users"));

    expect(confirmSpy).not.toHaveBeenCalled();
  });

  it("keeps save enabled and shows every error when submitting an invalid form", async () => {
    const handleSubmit = jest.fn().mockResolvedValue(true);
    const render = createTestRenderer();
    const { user } = render(
      <Sso
        appConfig={createMockConfig()}
        handleSubmit={handleSubmit}
        isPremiumTier
        isUpdatingSettings={false}
        router={({ push: noop } as unknown) as InjectedRouter}
        subsection="fleet-users"
      />
    );

    await user.click(screen.getByRole("checkbox", { name: "enableSso" }));

    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeEnabled();

    await user.click(save);

    expect(handleSubmit).not.toHaveBeenCalled();
    expect(
      screen.getByText("Enter an identity provider name")
    ).toBeInTheDocument();
    expect(screen.getByText("Enter an entity ID")).toBeInTheDocument();
    expect(save).toBeEnabled();
  });
});

describe("Sso - End users", () => {
  it("does not prompt on tab switch when a field is edited back to its original value", async () => {
    const { user, router } = renderSso("end-users");

    const idpName = screen.getByLabelText("Identity provider name");
    await user.type(idpName, "Okta");
    await user.clear(idpName);

    await user.click(screen.getByText("Fleet users"));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(router.push).toHaveBeenCalled();
  });

  it("prompts on tab switch when the form has unsaved changes", async () => {
    const { user } = renderSso("end-users");

    await user.type(screen.getByLabelText("Identity provider name"), "Okta");
    await user.click(screen.getByText("Fleet users"));

    expect(confirmSpy).toHaveBeenCalled();
  });
});
