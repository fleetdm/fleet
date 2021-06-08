import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import AppConfigForm from "components/forms/admin/AppConfigForm";
import { itBehavesLikeAFormInputElement } from "test/helpers";

describe("AppConfigForm - form", () => {
  const defaultProps = {
    formData: { org_name: "Kolide" },
    handleSubmit: noop,
    smtpConfigured: false,
    enrollSecret: [
      { secret: "foo_secret" },
      { secret: "bar_secret" },
      { secret: "baz_secret" },
    ],
  };
  const form = mount(<AppConfigForm {...defaultProps} />);

  describe("Organization Name input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "org_name");
    });
  });

  describe("Organization Avatar input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "org_logo_url");
    });
  });

  describe("Fleet App URL input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "server_url");
    });
  });

  describe("Sender Address input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "sender_address");
    });
  });

  describe("SMTP Server input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "server");
    });
  });

  describe("Port input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "port");
    });
  });

  describe("Enable SSL/TLS input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "enable_ssl_tls", "Checkbox");
    });
  });

  describe("SMTP user name input", () => {
    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "user_name");
    });
  });

  describe("SMTP user password input", () => {
    it("renders an HTML password input", () => {
      const passwordField = form.find('input[name="password"]');

      expect(passwordField.prop("type")).toEqual("password");
    });

    it("renders an input field", () => {
      itBehavesLikeAFormInputElement(form, "password");
    });
  });

  describe("Enroll Secret", () => {
    it("renders enroll secrets table", () => {
      expect(form.find("EnrollSecretTable").length).toEqual(1);
    });
  });

  it("does not render advanced options by default", () => {
    expect(form.find({ name: "domain" }).length).toEqual(0);
    expect(form.find("Slider").length).toEqual(0);
  });

  describe("Advanced options", () => {
    beforeAll(() => {
      form.find(".app-config-form__show-options").simulate("click");
    });

    it('renders advanced options when "Advanced Options" is clicked', () => {
      expect(form.find({ name: "domain" }).hostNodes().length).toEqual(1);
      expect(form.find("Slider").length).toEqual(4);
    });

    it("disables host expiry window by default", () => {
      const InputField = form.find({ name: "host_expiry_window" });
      const inputElement = InputField.find("input");
      expect(inputElement.length).toEqual(1);
      expect(inputElement.hasClass("input-field--disabled")).toBe(true);
    });

    it("enables host expiry window", () => {
      form
        .find({ name: "host_expiry_enabled" })
        .find("button")
        .simulate("click");
      const InputField = form.find({ name: "host_expiry_window" });
      const inputElement = InputField.find("input");
      expect(inputElement.hasClass("input-field--disabled")).toBe(false);
    });

    it("renders live query disabled input", () => {
      form.find({ name: "live_query_disabled" });
      expect(form.length).toEqual(1);
    });
  });
});
