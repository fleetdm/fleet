import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { fillInFormInput } from "test/helpers";
import targetMock from "test/target_mock";
import QueryForm from "./index";

const query = {
  id: 1,
  name: "All users",
  description: "Query to get all users",
  query: "SELECT * FROM users",
};
const queryText = "SELECT * FROM users";

const defaultProps = {
  handleSubmit: noop,
  onTargetSelect: noop,
  onOsqueryTableSelect: noop,
  onRunQuery: noop,
  onUpdate: noop,
};

describe("QueryForm - component", () => {
  beforeEach(targetMock);

  it("renders the base error", () => {
    const baseError = "Unable to authenticate the current user";
    const formWithError = mount(
      <QueryForm {...defaultProps} serverErrors={{ base: baseError }} />
    );
    const formWithoutError = mount(<QueryForm {...defaultProps} />);

    expect(formWithError.text()).toContain(baseError);
    expect(formWithoutError.text()).not.toContain(baseError);
  });

  it("renders InputFields for the query name and description", () => {
    const form = mount(
      <QueryForm
        {...defaultProps}
        query={query}
        queryText={queryText}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");

    expect(inputFields.length).toEqual(2);
    expect(inputFields.find({ name: "name" }).length).toBeGreaterThan(0);
    expect(inputFields.find({ name: "description" }).length).toBeGreaterThan(0);
  });

  it("validates the query name before saving changes", () => {
    const updateSpy = jest.fn();
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        onUpdate={updateSpy}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const nameInput = inputFields.find({ name: "name" });

    fillInFormInput(nameInput, "");

    const saveDropButton = form.find(".query-form__save").hostNodes();

    saveDropButton.simulate("click");
    form.find("li").first().find("Button").simulate("click");

    expect(updateSpy).not.toHaveBeenCalled();
  });

  it("calls the handleSubmit prop when the form is valid", () => {
    const spy = jest.fn();
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        onUpdate={spy}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const nameInput = inputFields.find({ name: "name" });

    fillInFormInput(nameInput, "New query name");

    const saveDropButton = form.find(".query-form__save").hostNodes();

    saveDropButton.simulate("click");
    form.find("li").first().find("Button").simulate("click");

    expect(spy).toHaveBeenCalledWith({
      description: query.description,
      name: "New query name",
      query: queryText,
    });
  });

  it("enables the Save Changes button when the name input changes", () => {
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const nameInput = inputFields.find('input[name="name"]');
    let saveChangesOption = form
      .find("li.dropdown-button__option")
      .first()
      .find("Button");

    expect(saveChangesOption.props()).toMatchObject({
      disabled: true,
    });

    fillInFormInput(nameInput, "New query name");
    nameInput.simulate("change", { target: { value: "New query name" } });

    saveChangesOption = form
      .find("li.dropdown-button__option")
      .first()
      .find("Button");
    expect(saveChangesOption.props()).not.toMatchObject({
      disabled: true,
    });
  });

  it("enables the Save Changes button when the description input changes", () => {
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const descriptionInput = inputFields.find({ name: "description" });
    let saveChangesOption = form
      .find("li.dropdown-button__option")
      .first()
      .find("Button");

    expect(saveChangesOption.props()).toMatchObject({
      disabled: true,
    });

    fillInFormInput(descriptionInput, "New query description");

    saveChangesOption = form
      .find("li.dropdown-button__option")
      .first()
      .find("Button");
    expect(saveChangesOption.props()).not.toMatchObject({
      disabled: true,
    });
  });

  it('calls the onSaveAsNew prop when "Save As New" is clicked and the form is valid', () => {
    const onSaveAsNewSpy = jest.fn();
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        handleSubmit={onSaveAsNewSpy}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const nameInput = inputFields.find({ name: "name" });
    const saveAsNewOption = form
      .find("li.dropdown-button__option")
      .last()
      .find("Button");

    fillInFormInput(nameInput, "New query name");

    saveAsNewOption.simulate("click");

    expect(onSaveAsNewSpy).toHaveBeenCalled();
    expect(onSaveAsNewSpy).toHaveBeenCalledWith({
      ...query,
      name: "New query name",
      query: queryText,
    });
  });

  it('does not call the onSaveAsNew prop when "Save As New" is clicked and the form is not valid', () => {
    const onSaveAsNewSpy = jest.fn();
    const form = mount(
      <QueryForm
        {...defaultProps}
        formData={{ ...query, query: queryText }}
        handleSubmit={onSaveAsNewSpy}
        hasSavePermissions
      />
    );
    const inputFields = form.find("InputField");
    const nameInput = inputFields.find({ name: "name" });
    const saveAsNewOption = form
      .find("li.dropdown-button__option")
      .last()
      .find("Button");

    fillInFormInput(nameInput, "");

    saveAsNewOption.simulate("click");

    expect(onSaveAsNewSpy).not.toHaveBeenCalled();
    expect(form.state()).toMatchObject({
      errors: {
        name: "Query name must be present",
      },
    });
  });
});
