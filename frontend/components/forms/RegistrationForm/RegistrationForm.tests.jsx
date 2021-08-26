import React from "react";
import { mount } from "enzyme";

import RegistrationForm from "components/forms/RegistrationForm";

describe("RegistrationForm - component", () => {
  it("renders AdminDetails and header on the first page", () => {
    const form = mount(<RegistrationForm page={1} />);

    expect(form.find("AdminDetails").length).toEqual(1);
    expect(form.text()).toContain("Setup user");
  });

  it("renders OrgDetails on the second page", () => {
    const form = mount(<RegistrationForm page={2} />);

    expect(form.find("OrgDetails").length).toEqual(1);
    expect(form.text()).toContain("Organization details");
  });

  it("renders FleetDetails on the third page", () => {
    const form = mount(<RegistrationForm page={3} />);

    expect(form.find("FleetDetails").length).toEqual(1);
    expect(form.text()).toContain("Set Fleet URL");
  });

  it("renders ConfirmationPage on the fourth page", () => {
    const form = mount(<RegistrationForm page={4} />);

    expect(form.find("ConfirmationPage").length).toEqual(1);
    expect(form.text()).toContain("Success");
  });
});
