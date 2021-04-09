import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import EditPackForm from "components/forms/packs/EditPackForm";
import { fillInFormInput } from "test/helpers";
import { packStub } from "test/stubs";

describe("EditPackForm - component", () => {
  describe("form fields", () => {
    const form = mount(
      <EditPackForm formData={packStub} handleSubmit={noop} onCancel={noop} />
    );

    it("has the correct form fields", () => {
      expect(form.find("InputField").length).toEqual(2);
      expect(form.find("SelectTargetsDropdown").length).toEqual(1);
      expect(form.find("Button").length).toEqual(2);
    });
  });

  describe("form submission", () => {
    it("submits the forms with the form data", () => {
      const spy = jest.fn();
      const form = mount(
        <EditPackForm formData={packStub} handleSubmit={spy} onCancel={noop} />
      ).find(".edit-pack-form");

      const nameInput = form.find({ name: "name" }).find("input");
      const descriptionInput = form
        .find({ name: "description" })
        .find("textarea");

      fillInFormInput(nameInput, "Updated pack name");
      fillInFormInput(descriptionInput, "Updated pack description");
      form.simulate("submit");

      expect(spy).toHaveBeenCalledWith({
        ...packStub,
        name: "Updated pack name",
        description: "Updated pack description",
      });
    });

    it('calls the onCancel prop when "Cancel" is clicked', () => {
      const spy = jest.fn();
      const form = mount(
        <EditPackForm formData={packStub} handleSubmit={noop} onCancel={spy} />
      ).find(".edit-pack-form");
      const cancelBtn = form.find("Button").find({ children: "Cancel" });

      cancelBtn.first().simulate("click");

      expect(spy).toHaveBeenCalled();
    });
  });
});
