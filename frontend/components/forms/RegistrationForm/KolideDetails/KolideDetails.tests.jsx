import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import KolideDetails from "components/forms/RegistrationForm/KolideDetails";
import { fillInFormInput } from "test/helpers";

describe("KolideDetails - form", () => {
  describe("kolide web address input", () => {
    it("renders an input field", () => {
      const form = mount(<KolideDetails handleSubmit={noop} />);
      const kolideWebAddressField = form.find({ name: "kolide_server_url" });

      expect(kolideWebAddressField.length).toBeGreaterThan(0);
    });

    it("updates state when the field changes", () => {
      const form = mount(<KolideDetails handleSubmit={noop} />);
      const kolideWebAddressField = form
        .find({ name: "kolide_server_url" })
        .find("input");

      fillInFormInput(kolideWebAddressField, "https://gnar.kolide.co");

      expect(form.state().formData).toMatchObject({
        kolide_server_url: "https://gnar.kolide.co",
      });
    });
  });

  describe("submitting the form", () => {
    it("validates the presence of the kolide web address field", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const htmlForm = form.find("form");

      htmlForm.simulate("submit");

      expect(handleSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        kolide_server_url: "Kolide web address must be completed",
      });
    });

    it("validates the kolide web address field starts with https://", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const kolideWebAddressField = form
        .find({ name: "kolide_server_url" })
        .find("input");
      const htmlForm = form.find("form");

      fillInFormInput(kolideWebAddressField, "http://gnar.kolide.co");
      htmlForm.simulate("submit");

      expect(handleSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        kolide_server_url: "Kolide web address must start with https://",
      });
    });

    it("submits the form when valid", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<KolideDetails handleSubmit={handleSubmitSpy} />);
      const kolideWebAddressField = form
        .find({ name: "kolide_server_url" })
        .find("input");
      const htmlForm = form.find("form");

      fillInFormInput(kolideWebAddressField, "https://gnar.kolide.co");
      htmlForm.simulate("submit");

      expect(handleSubmitSpy).toHaveBeenCalled();
    });
  });
});
