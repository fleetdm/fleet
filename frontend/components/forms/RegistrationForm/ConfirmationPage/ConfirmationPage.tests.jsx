import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ConfirmationPage from "components/forms/RegistrationForm/ConfirmationPage";

describe("ConfirmationPage - form", () => {
  const formData = {
    name: "Rachel Perkins",
    email: "rachel@fleet.com",
    org_name: "Fleet",
    server_url: "http://localhost:8080",
  };

  it("renders the user information", () => {
    const form = mount(
      <ConfirmationPage formData={formData} handleSubmit={noop} />
    );

    expect(form.text()).toContain(formData.name);
    expect(form.text()).toContain(formData.email);
    expect(form.text()).toContain(formData.org_name);
    expect(form.text()).toContain(formData.server_url);
  });

  it("submits the form", () => {
    const handleSubmitSpy = jest.fn();
    const form = mount(
      <ConfirmationPage formData={formData} handleSubmit={handleSubmitSpy} />
    );
    const htmlForm = form.find("form");

    htmlForm.simulate("submit");

    expect(handleSubmitSpy).toHaveBeenCalled();
  });
});
