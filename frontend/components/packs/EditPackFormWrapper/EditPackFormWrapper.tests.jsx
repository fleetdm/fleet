import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import EditPackFormWrapper from "components/packs/EditPackFormWrapper";
import { packStub } from "test/stubs";

describe("EditPackFormWrapper - component", () => {
  it("does not render the EditPackForm by default", () => {
    const component = mount(
      <EditPackFormWrapper
        isEdit={false}
        onCancelEditPack={noop}
        onEditPack={noop}
        pack={packStub}
      />
    );

    expect(component.find("EditPackForm").length).toEqual(0);
  });

  it("renders the EditPackForm when isEdit is true", () => {
    const component = mount(
      <EditPackFormWrapper
        isEdit
        onCancelEditPack={noop}
        onEditPack={noop}
        pack={packStub}
      />
    );

    expect(component.find("EditPackForm").length).toEqual(1);
  });

  it("calls onEditPack when EDIT is clicked", () => {
    const spy = jest.fn();
    const component = mount(
      <EditPackFormWrapper
        isEdit={false}
        onCancelEditPack={noop}
        onEditPack={spy}
        pack={packStub}
      />
    );
    const editBtn = component.find("Button");

    editBtn.simulate("click");

    expect(spy).toHaveBeenCalled();
  });
});
