import React from "react";
import { find } from "lodash";
import { mount } from "enzyme";

import ConnectedAllPacksPage, {
  AllPacksPage,
} from "pages/packs/AllPacksPage/AllPacksPage";
import {
  connectedComponent,
  fillInFormInput,
  reduxMockStore,
} from "test/helpers";
import packActions from "redux/nodes/entities/packs/actions";
import { packStub } from "test/stubs";
import scheduledQueryActions from "redux/nodes/entities/scheduled_queries/actions";

const store = {
  entities: {
    packs: {
      loading: false,
      data: {
        [packStub.id]: packStub,
        101: {
          ...packStub,
          id: 101,
          name: "My unique pack name",
        },
      },
    },
  },
};

describe("AllPacksPage - component", () => {
  beforeEach(() => {
    jest
      .spyOn(packActions, "loadAll")
      .mockImplementation(() => () => Promise.resolve([]));
  });

  describe("rendering", () => {
    it("does not render when packs are loading", () => {
      const page = mount(<AllPacksPage loadingPacks packs={[packStub]} />);

      expect(page.html()).toBeFalsy();
    });

    it("renders a PacksList component", () => {
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore: reduxMockStore(store),
      });
      const page = mount(Component);

      expect(page.find("PacksList").length).toEqual(1);
    });

    it("renders the PackInfoSidePanel by default", () => {
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore: reduxMockStore(store),
      });
      const page = mount(Component);

      expect(page.find("PackInfoSidePanel").length).toEqual(1);
    });
  });

  it("filters the packs list", () => {
    const Component = connectedComponent(ConnectedAllPacksPage, {
      mockStore: reduxMockStore(store),
    });
    const page = mount(Component).find("AllPacksPage");
    const packsFilterInput = page.find({ name: "pack-filter" }).find("input");

    expect(page.instance().getPacks().length).toEqual(2);

    fillInFormInput(packsFilterInput, "My unique pack name");

    expect(page.instance().getPacks().length).toEqual(1);
  });

  it("updates checkedPackIDs in state when the select all packs Checkbox is toggled", () => {
    const page = mount(<AllPacksPage packs={[packStub]} />);
    let selectAllPacks = page.find({ name: "select-all-packs" }).hostNodes();

    expect(page.state("checkedPackIDs")).toEqual([]);

    selectAllPacks.simulate("change");

    expect(page.state("checkedPackIDs")).toEqual([packStub.id]);

    selectAllPacks = page.find({ name: "select-all-packs" }).hostNodes();
    selectAllPacks.simulate("change");

    expect(page.state("checkedPackIDs")).toEqual([]);
  });

  it("updates checkedPackIDs in state when a pack row Checkbox is toggled", () => {
    const page = mount(<AllPacksPage packs={[packStub]} />);
    let selectPack = page
      .find({ name: `select-pack-${packStub.id}` })
      .hostNodes();

    expect(page.state("checkedPackIDs")).toEqual([]);

    selectPack.simulate("change");

    expect(page.state("checkedPackIDs")).toEqual([packStub.id]);

    selectPack = page.find({ name: `select-pack-${packStub.id}` }).hostNodes();
    selectPack.simulate("change");

    expect(page.state("checkedPackIDs")).toEqual([]);
  });

  describe("bulk actions", () => {
    const packs = [
      packStub,
      { ...packStub, id: 101, name: "My unique pack name" },
    ];

    it("displays the bulk action buttons when a pack is checked", () => {
      const page = mount(<AllPacksPage packs={packs} />);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      selectAllPacks.hostNodes().simulate("change");

      expect(page.state("checkedPackIDs")).toEqual([packStub.id, 101]);
      expect(
        page.find(".all-packs-page__bulk-action-btn--disable").length
      ).toBeGreaterThan(0);
      expect(
        page.find(".all-packs-page__bulk-action-btn--enable").length
      ).toBeGreaterThan(0);
      expect(
        page.find(".all-packs-page__bulk-action-btn--delete").length
      ).toBeGreaterThan(0);
    });

    it("dispatches the pack update function when disable is clicked", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      selectAllPacks.hostNodes().simulate("change");

      const disableBtn = page
        .find(".all-packs-page__bulk-action-btn--disable")
        .hostNodes();

      disableBtn.simulate("click");

      const dispatchedActions = mockStore.getActions();

      expect(dispatchedActions).toContainEqual({
        type: "packs_UPDATE_REQUEST",
      });
    });

    it("dispatches the pack update function when enable is clicked", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      selectAllPacks.hostNodes().simulate("change");

      const enableBtn = page
        .find(".all-packs-page__bulk-action-btn--enable")
        .hostNodes();

      enableBtn.simulate("click");

      const dispatchedActions = mockStore.getActions();

      expect(dispatchedActions).toContainEqual({
        type: "packs_UPDATE_REQUEST",
      });
    });

    it("loads a modal when delete is clicked", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      expect(page.find("Modal").length).toEqual(0);

      selectAllPacks.hostNodes().simulate("change");

      const deleteBtn = page
        .find(".all-packs-page__bulk-action-btn--delete")
        .hostNodes();

      deleteBtn.simulate("click");

      expect(page.find("Modal").length).toEqual(1);
    });

    it("dispatches the pack destroy action when the modal is confirmed", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      expect(page.find("Modal").length).toEqual(0);

      selectAllPacks.hostNodes().simulate("change");

      const deleteBtn = page
        .find(".all-packs-page__bulk-action-btn--delete")
        .hostNodes();

      deleteBtn.simulate("click");

      const modal = page.find("Modal");

      modal.find("Button").first().simulate("click");

      const dispatchedActions = mockStore.getActions();

      expect(dispatchedActions).toContainEqual({
        type: "packs_DESTROY_REQUEST",
      });
    });

    it("does not dispatch the pack destroy action when the modal is canceled", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component);
      const selectAllPacks = page.find({ name: "select-all-packs" });

      expect(page.find("Modal").length).toEqual(0);

      selectAllPacks.hostNodes().simulate("change");

      const deleteBtn = page
        .find(".all-packs-page__bulk-action-btn--delete")
        .hostNodes();

      deleteBtn.simulate("click");

      const modal = page.find("Modal");

      modal.find("Button").last().simulate("click");

      const dispatchedActions = mockStore.getActions();

      expect(dispatchedActions).not.toContainEqual({
        type: "packs_DESTROY_REQUEST",
      });
    });
  });

  describe("selecting a pack", () => {
    it("updates the URL when a pack is selected", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
      });
      const page = mount(Component).find("AllPacksPage");
      const firstRow = page.find("Row").last();

      expect(page.prop("selectedPack")).toBeFalsy();

      firstRow.find("ClickableTableRow").last().simulate("click");

      const dispatchedActions = mockStore.getActions();
      const locationChangeAction = find(dispatchedActions, {
        type: "@@router/CALL_HISTORY_METHOD",
      });

      expect(locationChangeAction.payload.args).toEqual([
        {
          pathname: "/packs/manage",
          query: { selectedPack: packStub.id },
        },
      ]);
    });

    it("sets the selectedPack prop", () => {
      jest
        .spyOn(scheduledQueryActions, "loadAll")
        .mockImplementation(() => () => Promise.resolve([]));

      const mockStore = reduxMockStore(store);
      const props = { location: { query: { selectedPack: packStub.id } } };
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
        props,
      });
      const page = mount(Component).find("AllPacksPage");

      expect(page.prop("selectedPack")).toEqual(packStub);
    });

    it("goes to the edit pack edit page when table row is double clicked", () => {
      const mockStore = reduxMockStore(store);
      const props = { location: { query: { selectedPack: packStub.id } } };
      const Component = connectedComponent(ConnectedAllPacksPage, {
        mockStore,
        props,
      });
      const page = mount(Component).find("AllPacksPage");
      const tableRow = page.find("PacksList").find(".packs-list-row--selected");

      tableRow.hostNodes().simulate("doubleclick");

      const routerChangeAction = find(mockStore.getActions(), {
        type: "@@router/CALL_HISTORY_METHOD",
      });

      expect(routerChangeAction.payload).toEqual({
        method: "push",
        args: [`/packs/${packStub.id}`],
      });
    });
  });
});
