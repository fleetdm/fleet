import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import UserRow from "components/UserRow/UserRow";
import { fillInFormInput } from "test/helpers";
import { userStub } from "test/stubs";

describe("UserRow - component", () => {
  const defaultInviteProps = {
    isCurrentUser: false,
    isEditing: false,
    isInvite: true,
    onEditUser: noop,
    onSelect: noop,
    onToggleEditUser: noop,
    userErrors: {},
  };

  const defaultUserProps = {
    ...defaultInviteProps,
    isInvite: false,
    user: userStub,
  };

  it("renders a user row", () => {
    const props = { ...defaultUserProps, user: userStub };
    const component = mount(<UserRow {...props} />);

    expect(component.length).toEqual(1);
    expect(component.find("Dropdown").length).toEqual(1);
  });

  it("calls the onToggleEditUser prop with the user when Modify Details is selected", () => {
    const spy = jest.fn();
    const props = { ...defaultUserProps, onToggleEditUser: spy };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component.find('[aria-label="Modify Details"]').simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(userStub);
  });

  it("calls the onSelect prop with the user when Promote User is selected", () => {
    const spy = jest.fn();
    const props = { ...defaultUserProps, onSelect: spy };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component.find('[aria-label="Promote User"]').simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(userStub, "promote_user");
  });

  it("calls the onSelect prop with the user when Demote User is selected", () => {
    const adminUser = { ...userStub, admin: true };
    const spy = jest.fn();
    const props = { ...defaultUserProps, onSelect: spy, user: adminUser };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component.find('[aria-label="Demote User"]').simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(adminUser, "demote_user");
  });

  it("calls the onSelect prop with the user when Disable Account is selected", () => {
    const spy = jest.fn();
    const props = { ...defaultUserProps, onSelect: spy };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component.find('[aria-label="Disable Account"]').simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(userStub, "disable_account");
  });

  it("calls the onSelect prop with the user when Enable Account is selected", () => {
    const disabledUser = { ...userStub, enabled: false };
    const spy = jest.fn();
    const props = { ...defaultUserProps, onSelect: spy, user: disabledUser };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component.find('[aria-label="Enable Account"]').simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(disabledUser, "enable_account");
  });

  it("calls the onSelect prop with the user when Require Password Reset is selected", () => {
    const spy = jest.fn();
    const props = { ...defaultUserProps, onSelect: spy };
    const component = mount(<UserRow {...props} />);

    component.find(".Select-control").simulate("keyDown", { keyCode: 40 });
    component
      .find('[aria-label="Require Password Reset"]')
      .simulate("mousedown");

    expect(spy).toHaveBeenCalledWith(userStub, "reset_password");
  });

  it("calls the onEditUser prop with the user and updated user when the edit form is submitted", () => {
    const spy = jest.fn();
    const props = { ...defaultUserProps, isEditing: true, onEditUser: spy };
    const component = mount(<UserRow {...props} />);
    const form = component.find("EditUserForm");

    expect(form.length).toEqual(1);

    const nameInput = form.find({ name: "name" }).find("input");

    fillInFormInput(nameInput, "Foobar");
    form.simulate("submit");

    expect(spy).toHaveBeenCalledWith(userStub, { ...userStub, name: "Foobar" });
  });
});
