import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import DefaultConfigurePackQueryForm, {
  ConfigurePackQueryForm,
} from "components/forms/ConfigurePackQueryForm/ConfigurePackQueryForm";
import {
  itBehavesLikeAFormDropdownElement,
  itBehavesLikeAFormInputElement,
} from "test/helpers";
import { scheduledQueryStub } from "test/stubs";

describe("ConfigurePackQueryForm - component", () => {
  describe("form fields", () => {
    const form = mount(<DefaultConfigurePackQueryForm handleSubmit={noop} />);

    it("updates form state", () => {
      itBehavesLikeAFormInputElement(form, "interval");
      itBehavesLikeAFormDropdownElement(form, "logging_type");
      itBehavesLikeAFormDropdownElement(form, "platform");
      itBehavesLikeAFormDropdownElement(form, "version");
      itBehavesLikeAFormInputElement(form, "shard");
    });
  });

  describe("platform options", () => {
    const onChangeSpy = jest.fn();
    const fieldsObj = {
      platform: {
        onChange: onChangeSpy,
      },
      interval: {},
      logging_type: {},
      version: {},
    };
    const form = mount(
      <ConfigurePackQueryForm
        fields={fieldsObj}
        handleSubmit={noop}
        formData={{ query_id: 1 }}
      />
    );

    it("doesn't allow All when other options are chosen", () => {
      form.instance().handlePlatformChoice(",windows");

      expect(onChangeSpy).toHaveBeenCalledWith("windows");
    });

    it("doesn't allow other options when All is chosen", () => {
      form.instance().handlePlatformChoice("darwin,linux,");

      expect(onChangeSpy).toHaveBeenCalledWith("");
    });
  });

  describe("submitting the form", () => {
    const spy = jest.fn();
    const form = mount(
      <DefaultConfigurePackQueryForm
        handleSubmit={spy}
        formData={{ query_id: 1 }}
      />
    );

    it("submits the form with the form data", () => {
      itBehavesLikeAFormInputElement(form, "interval", "InputField", 123);
      itBehavesLikeAFormDropdownElement(form, "logging_type");
      itBehavesLikeAFormDropdownElement(form, "platform");
      itBehavesLikeAFormDropdownElement(form, "version");
      itBehavesLikeAFormInputElement(form, "shard", "InputField", 12);

      form.find("form").simulate("submit");

      expect(spy).toHaveBeenCalledWith({
        interval: 123,
        logging_type: "snapshot",
        platform: "",
        query_id: 1,
        version: "",
        shard: 12,
      });
    });
  });

  describe("cancelling the form", () => {
    const CancelButton = (form) =>
      form.find(".configure-pack-query-form__cancel-btn");

    it("displays a cancel Button when updating a scheduled query", () => {
      const NewScheduledQueryForm = mount(
        <DefaultConfigurePackQueryForm
          formData={{ query_id: 1 }}
          handleSubmit={noop}
          onCancel={noop}
        />
      );
      const UpdateScheduledQueryForm = mount(
        <DefaultConfigurePackQueryForm
          formData={scheduledQueryStub}
          handleSubmit={noop}
          onCancel={noop}
        />
      );

      expect(CancelButton(NewScheduledQueryForm).length).toEqual(0);
      expect(CancelButton(UpdateScheduledQueryForm).length).toBeGreaterThan(0);
    });

    it("calls the onCancel prop when the cancel Button is clicked", () => {
      const spy = jest.fn();
      const UpdateScheduledQueryForm = mount(
        <DefaultConfigurePackQueryForm
          formData={scheduledQueryStub}
          handleSubmit={noop}
          onCancel={spy}
        />
      );

      CancelButton(UpdateScheduledQueryForm).hostNodes().simulate("click");

      expect(spy).toHaveBeenCalledWith(scheduledQueryStub);
    });
  });
});
