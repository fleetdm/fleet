import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import PacksList from "components/packs/PacksList";
import { packStub } from "test/stubs";

describe("PacksList - component", () => {
  const props = {
    onCheckAllPacks: noop,
    onCheckPack: noop,
    onSelectPack: noop,
  };

  it("renders", () => {
    expect(mount(<PacksList {...props} packs={[packStub]} />).length).toEqual(
      1
    );
  });

  it("calls the onCheckAllPacks prop when select all packs checkbox is checked", () => {
    const spy = jest.fn();
    const component = mount(
      <PacksList {...props} onCheckAllPacks={spy} packs={[packStub]} />
    );

    component.find({ name: "select-all-packs" }).hostNodes().simulate("change");

    expect(spy).toHaveBeenCalledWith(true);
  });

  it("calls the onCheckPack prop when a pack checkbox is checked", () => {
    const spy = jest.fn();
    const component = mount(
      <PacksList {...props} onCheckPack={spy} packs={[packStub]} />
    );
    const packCheckbox = component.find({ name: `select-pack-${packStub.id}` });

    packCheckbox.hostNodes().simulate("change");

    expect(spy).toHaveBeenCalledWith(true, packStub.id);
  });
});
