import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockRouter } from "test/test-utils";

import createMockConfig, { createMockMdmConfig } from "__mocks__/configMock";

import { ScepCertificateContent } from "./ScepPage";

const FORM_DATA = { scepUrl: "", adminUrl: "", username: "", password: "" };

describe("Scep Page", () => {
  it("renders PremiumFeatureMessage for non-premium tier", () => {
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={FORM_DATA}
        formErrors={{}}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig()}
        isLoading={false}
        isSaving={false}
        showDataError={false}
        isPremiumTier={false} // test
      />
    );
    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
  });
  it("renders TurnOnMdmMessage when MDM is not enabled", () => {
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={FORM_DATA}
        formErrors={{}}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig({
          mdm: createMockMdmConfig({ enabled_and_configured: false }),
        })}
        isLoading={false}
        isSaving={false}
        showDataError={false}
        isPremiumTier
      />
    );
    expect(screen.getByText("Turn on Apple MDM")).toBeInTheDocument();
  });
  it("renders Spinner when loading", () => {
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={FORM_DATA}
        formErrors={{}}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig()}
        isLoading // test
        isSaving={false}
        showDataError={false}
        isPremiumTier
      />
    );
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });
  it("renders DataError when showDataError is true", () => {
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={{ scepUrl: "", adminUrl: "", username: "", password: "" }}
        formErrors={{}}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig()}
        isLoading={false}
        isSaving={false}
        showDataError // test
        isPremiumTier
      />
    );
    expect(screen.getByText("Something's gone wrong.")).toBeInTheDocument();
  });
  it("renders form fields correctly", () => {
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={FORM_DATA}
        formErrors={{}}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig()}
        isLoading={false}
        isSaving={false}
        showDataError={false}
        isPremiumTier
      />
    );
    expect(screen.getByLabelText("SCEP URL")).toBeInTheDocument();
    expect(screen.getByLabelText("Admin URL")).toBeInTheDocument();
    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });
  it("displays error messages for invalid inputs", () => {
    const FORM_ERRORS = { scepUrl: "Invalid URL", adminUrl: "Invalid URL" };
    const INVALID_FORM_DATA = {
      scepUrl: "invalid",
      adminUrl: "invalid",
      username: "",
      password: "",
    };
    render(
      <ScepCertificateContent
        router={createMockRouter()}
        onFormSubmit={jest.fn()}
        formData={INVALID_FORM_DATA}
        formErrors={FORM_ERRORS}
        onInputChange={jest.fn()}
        onBlur={jest.fn()}
        config={createMockConfig()}
        isLoading={false}
        isSaving={false}
        showDataError={false}
        isPremiumTier
      />
    );
    expect(screen.getAllByLabelText("Invalid URL").length).toBe(2);
  });
});
