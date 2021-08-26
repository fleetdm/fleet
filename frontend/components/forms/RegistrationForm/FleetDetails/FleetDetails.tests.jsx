import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import FleetDetails from "components/forms/RegistrationForm/FleetDetails";
import { fillInFormInput } from "test/helpers";

describe("FleetDetails - form", () => {
  describe("fleet web address input", () => {
    it("renders an input field", () => {
      const form = mount(<FleetDetails handleSubmit={noop} />);
      const fleetWebAddressField = form.find({ name: "server_url" });

      expect(fleetWebAddressField.length).toBeGreaterThan(0);
    });

    it("updates state when the field changes", () => {
      const form = mount(<FleetDetails handleSubmit={noop} />);
      const serverAddressField = form
        .find({ name: "server_url" })
        .find("input");

      fillInFormInput(serverAddressField, "https://gnar.Fleet.co");

      expect(form.state().formData).toMatchObject({
        server_url: "https://gnar.Fleet.co",
      });
    });
  });

  describe("submitting the form", () => {
    it("validates the presence of the fleet web address field", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<FleetDetails handleSubmit={handleSubmitSpy} />);
      const htmlForm = form.find("form");

      htmlForm.simulate("submit");

      expect(handleSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        server_url: "Fleet web address must be completed",
      });
    });

    it("validates the fleet web address field starts with https://", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<FleetDetails handleSubmit={handleSubmitSpy} />);
      const fleetWebAddressField = form
        .find({ name: "server_url" })
        .find("input");
      const htmlForm = form.find("form");

      fillInFormInput(fleetWebAddressField, "http://gnar.Fleet.co");
      htmlForm.simulate("submit");

      expect(handleSubmitSpy).not.toHaveBeenCalled();
      expect(form.state().errors).toMatchObject({
        server_url: "Fleet web address must start with https://",
      });
    });

    it("submits the form when valid", () => {
      const handleSubmitSpy = jest.fn();
      const form = mount(<FleetDetails handleSubmit={handleSubmitSpy} />);
      const fleetWebAddressField = form
        .find({ name: "server_url" })
        .find("input");
      const htmlForm = form.find("form");

      fillInFormInput(fleetWebAddressField, "https://gnar.Fleet.co");
      htmlForm.simulate("submit");

      expect(handleSubmitSpy).toHaveBeenCalled();
    });
  });
});
