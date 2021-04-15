import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import Row from "components/packs/PacksList/Row";
import { packStub } from "test/stubs";

describe("PacksList - Row - component", () => {
  it("renders", () => {
    expect(mount(<Row pack={packStub} />).length).toEqual(1);
  });

  it("calls the onCheck prop with the value and pack id when checked", () => {
    const spy = jest.fn();
    const component = mount(<Row checked onCheck={spy} pack={packStub} />);

    component
      .find({ name: `select-pack-${packStub.id}` })
      .hostNodes()
      .simulate("change");

    expect(spy).toHaveBeenCalledWith(false, packStub.id);
  });

  it("calls the onDoubleClick prop when double clicked", () => {
    const spy = jest.fn();
    const component = mount(<Row pack={packStub} onDoubleClick={spy} />);

    component.find("ClickableTableRow").simulate("doubleclick");

    expect(spy).toHaveBeenCalledWith(packStub);
  });

  it("outputs host count", () => {
    const packWithHosts = { ...packStub, total_hosts_count: 3 };
    const packWithoutHosts = { ...packStub, total_hosts_count: 0 };
    const componentWithHosts = mount(
      <Row checked onCheck={noop} pack={packWithHosts} />
    );
    const componentWithoutHosts = mount(
      <Row checked onCheck={noop} pack={packWithoutHosts} />
    );

    const hostCountWith = componentWithHosts.find(
      ".packs-list-row__td-host-count"
    );
    const hostCountWithout = componentWithoutHosts.find(
      ".packs-list-row__td-host-count"
    );
    expect(hostCountWith.text()).toEqual("3");
    expect(hostCountWithout.text()).toEqual("0");
  });
});
