import React from "react";
import { find, noop } from "lodash";
import { mount } from "enzyme";

import ConnectedManageQueriesPage, {
  ManageQueriesPage,
} from "pages/queries/ManageQueriesPage/ManageQueriesPage";
import {
  connectedComponent,
  fillInFormInput,
  reduxMockStore,
} from "test/helpers";
import queryActions from "redux/nodes/entities/queries/actions";
import { queryStub } from "test/stubs";

const store = {
  entities: {
    queries: {
      loading: false,
      data: {
        [queryStub.id]: queryStub,
        101: {
          ...queryStub,
          id: 101,
          name: "alpha query",
        },
      },
    },
  },
};

describe("ManageQueriesPage - component", () => {
  beforeEach(() => {
    jest
      .spyOn(queryActions, "loadAll")
      .mockImplementation(() => () => Promise.resolve([]));
  });

  describe("rendering", () => {
    it("does not render if queries are loading", () => {
      const loadingQueriesStore = {
        entities: { queries: { loading: true, data: {} } },
      };
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore: reduxMockStore(loadingQueriesStore),
      });
      const page = mount(Component);

      expect(page.html()).toBeFalsy();
    });

    it("renders a QueriesList component", () => {
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore: reduxMockStore(store),
      });
      const page = mount(Component);

      expect(page.find("QueriesList").length).toEqual(1);
    });

    it("renders the QueryDetailsSidePanel when a query is selected", () => {
      const mockStore = reduxMockStore(store);
      const props = { location: { query: { selectedQuery: queryStub.id } } };
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore,
        props,
      });
      const page = mount(Component).find("ManageQueriesPage");

      expect(page.find("QueryDetailsSidePanel").length).toEqual(1);
    });

    it("resets checkedQueryIDs after successfully deleting checked queries", (done) => {
      const fakeEvt = { preventDefault: noop };
      const props = {
        dispatch: () => Promise.resolve(),
        loadingQueries: false,
        queries: [queryStub],
      };
      const page = mount(<ManageQueriesPage {...props} />);

      page.setState({ checkedQueryIDs: [queryStub.id], showModal: true });
      page
        .instance()
        .onDeleteQueries(fakeEvt)
        .then(() => {
          expect(page.state("showModal")).toEqual(false);
          expect(page.state("checkedQueryIDs")).toEqual([]);
          done();
        })
        .catch(done);
    });

    it("does not reset checkedQueryIDs if deleting a checked query is  unsuccessful", (done) => {
      const fakeEvt = { preventDefault: noop };
      const props = {
        dispatch: () => Promise.reject(),
        loadingQueries: false,
        queries: [queryStub],
      };
      const page = mount(<ManageQueriesPage {...props} />);

      page.setState({ checkedQueryIDs: [queryStub.id], showModal: true });
      page
        .instance()
        .onDeleteQueries(fakeEvt)
        .then(() => {
          expect(page.state("showModal")).toEqual(false);
          expect(page.state("checkedQueryIDs")).toEqual([queryStub.id]);
          done();
        })
        .catch(done);
    });
  });

  it("filters the queries list", () => {
    const Component = connectedComponent(ConnectedManageQueriesPage, {
      mockStore: reduxMockStore(store),
    });
    const page = mount(Component).find("ManageQueriesPage");
    const queryFilterInput = page.find({ name: "query-filter" }).find("input");

    expect(page.instance().getQueries().length).toEqual(2);

    fillInFormInput(queryFilterInput, "alpha query");

    expect(page.instance().getQueries().length).toEqual(1);
  });

  it("updates checkedQueryIDs in state when the check all queries Checkbox is toggled", () => {
    const page = mount(<ManageQueriesPage queries={[queryStub]} />);
    const selectAllQueries = page.find({ name: "check-all-queries" });

    expect(page.state("checkedQueryIDs")).toEqual([]);

    selectAllQueries.hostNodes().simulate("change");

    expect(page.state("checkedQueryIDs")).toEqual([queryStub.id]);

    selectAllQueries.hostNodes().simulate("change");

    expect(page.state("checkedQueryIDs")).toEqual([]);
  });

  it("updates checkedQueryIDs in state when a query row Checkbox is toggled", () => {
    const page = mount(<ManageQueriesPage queries={[queryStub]} />);
    const queryCheckbox = page.find({ name: `query-checkbox-${queryStub.id}` });

    expect(page.state("checkedQueryIDs")).toEqual([]);

    queryCheckbox.hostNodes().simulate("change");

    expect(page.state("checkedQueryIDs")).toEqual([queryStub.id]);

    queryCheckbox.hostNodes().simulate("change");

    expect(page.state("checkedQueryIDs")).toEqual([]);
  });

  it("goes to the edit query page when the Edit/Run Query button is the side panel is clicked", () => {
    const mockStore = reduxMockStore(store);
    const props = { location: { query: { selectedQuery: queryStub.id } } };
    const Component = connectedComponent(ConnectedManageQueriesPage, {
      mockStore,
      props,
    });
    const page = mount(Component);
    const button = page.find("QueryDetailsSidePanel").find("Button");

    button.simulate("click");

    const routerChangeAction = find(mockStore.getActions(), {
      type: "@@router/CALL_HISTORY_METHOD",
    });

    expect(routerChangeAction.payload).toEqual({
      method: "push",
      args: [`/queries/${queryStub.id}`],
    });
  });

  it("goes to the edit query page when table row is double clicked", () => {
    const mockStore = reduxMockStore(store);
    const props = { location: { query: { selectedQuery: queryStub.id } } };
    const Component = connectedComponent(ConnectedManageQueriesPage, {
      mockStore,
      props,
    });
    const page = mount(Component);
    const tableRow = page
      .find("QueriesListRow")
      .find(".queries-list-row--selected");

    tableRow.hostNodes().simulate("doubleclick");

    const routerChangeAction = find(mockStore.getActions(), {
      type: "@@router/CALL_HISTORY_METHOD",
    });

    expect(routerChangeAction.payload).toEqual({
      method: "push",
      args: [`/queries/${queryStub.id}`],
    });
  });

  describe("bulk delete action", () => {
    const queries = [queryStub, { ...queryStub, id: 101, name: "alpha query" }];

    it("displays the delete action button when a query is checked", () => {
      const page = mount(<ManageQueriesPage queries={queries} />);
      const checkAllQueries = page.find({ name: "check-all-queries" });

      checkAllQueries.hostNodes().simulate("change");

      expect(page.state("checkedQueryIDs")).toEqual([queryStub.id, 101]);
      expect(
        page.find(".manage-queries-page__delete-queries-btn").length
      ).toBeGreaterThan(0);
    });

    it("calls the API to delete once the Modal has been accepted", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore,
      });
      const page = mount(Component);
      const checkAllQueries = page.find({ name: "check-all-queries" });

      expect(page.find("Modal").length).toEqual(0);

      checkAllQueries.hostNodes().simulate("change");

      const deleteBtn = page
        .find(".manage-queries-page__delete-queries-btn")
        .hostNodes();

      deleteBtn.simulate("click");

      expect(mockStore.getActions()).not.toContainEqual({
        type: "queries_DESTROY_REQUEST",
      });

      expect(page.find("Modal").length).toEqual(1);

      page.find("Modal").find("Button").first().simulate("click");

      expect(mockStore.getActions()).toContainEqual({
        type: "queries_DESTROY_REQUEST",
      });
    });

    it("does not call the API if the Modal is not accepted", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore,
      });
      const page = mount(Component);
      const checkAllQueries = page.find({ name: "check-all-queries" });

      expect(page.find("Modal").length).toEqual(0);

      checkAllQueries.hostNodes().simulate("change");

      const deleteBtn = page
        .find(".manage-queries-page__delete-queries-btn")
        .hostNodes();

      deleteBtn.simulate("click");

      expect(mockStore.getActions()).not.toContainEqual({
        type: "queries_DESTROY_REQUEST",
      });

      expect(page.find("Modal").length).toEqual(1);

      page.find("Modal").find("Button").last().simulate("click");

      expect(mockStore.getActions()).not.toContainEqual({
        type: "queries_DESTROY_REQUEST",
      });
    });
  });

  describe("selecting a query", () => {
    it("updates the URL when a query is selected", () => {
      const mockStore = reduxMockStore(store);
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore,
      });
      const page = mount(Component).find("ManageQueriesPage");
      const firstRow = page.find("QueriesListRow").last();

      expect(page.prop("selectedQuery")).toBeFalsy();

      firstRow.find("ClickableTableRow").last().simulate("click");

      const dispatchedActions = mockStore.getActions();
      const locationChangeAction = find(dispatchedActions, {
        type: "@@router/CALL_HISTORY_METHOD",
      });

      expect(locationChangeAction.payload.args).toEqual([
        {
          pathname: "/queries/manage",
          query: { selectedQuery: queryStub.id },
        },
      ]);
    });

    it("sets the selectedQuery prop", () => {
      const mockStore = reduxMockStore(store);
      const props = { location: { query: { selectedQuery: queryStub.id } } };
      const Component = connectedComponent(ConnectedManageQueriesPage, {
        mockStore,
        props,
      });
      const page = mount(Component).find("ManageQueriesPage");

      expect(page.prop("selectedQuery")).toEqual(queryStub);
    });
  });
});
