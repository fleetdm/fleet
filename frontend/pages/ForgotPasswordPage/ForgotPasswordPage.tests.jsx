import React from "react";
import { mount } from "enzyme";

import { ForgotPasswordPage } from "./ForgotPasswordPage";

describe("ForgotPasswordPage - component", () => {
  it("renders the ForgotPasswordForm when there is no email prop", () => {
    const page = mount(<ForgotPasswordPage />);

    expect(page.find("ForgotPasswordForm").length).toEqual(1);
  });

  it("renders the email sent text when the email prop is present", () => {
    const email = "hi@thegnar.co";
    const page = mount(<ForgotPasswordPage email={email} />);

    expect(page.find("ForgotPasswordForm").length).toEqual(0);
    expect(page.text()).toContain(`An email was sent to ${email}.`);
  });
});
