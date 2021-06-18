import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import ConfirmationPage from "components/forms/RegistrationForm/ConfirmationPage";

describe("ConfirmationPage - form", () => {
  const formData = {
    username: "jmeller",
    email: "jason@Fleet.co",
    org_name: "Kolide",
    fleet_web_address: "http://Fleet.Fleet.co",
  };

  it("renders the user information", () => {
    const form = mount(
      <ConfirmationPage formData={formData} handleSubmit={noop} />
    );

    expect(form.text()).toContain(formData.username);
    expect(form.text()).toContain(formData.email);
    expect(form.text()).toContain(formData.org_name);
    expect(form.text()).toContain(formData.fleet_web_address);
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
