import React from "react";
import FileSave from "file-saver";
import { mount } from "enzyme";
import { noop } from "lodash";

import convertToCSV from "utilities/convert_to_csv";
import * as queryPageActions from "redux/nodes/components/QueryPages/actions";
import helpers from "test/helpers";
import hostActions from "redux/nodes/entities/hosts/actions";
import queryActions from "redux/nodes/entities/queries/actions";
import ConnectedQueryPage, {
  QueryPage,
} from "pages/queries/QueryPage/QueryPage";
import {
  hostStub,
  queryStub,
  labelStub,
  adminUserStub,
  configStub,
} from "test/stubs";

const {
  connectedComponent,
  createAceSpy,
  fillInFormInput,
  reduxMockStore,
} = helpers;
const { defaultSelectedOsqueryTable } = queryPageActions;
const locationProp = { params: {}, location: { query: {} } };

describe("QueryPage - component", () => {
  beforeEach(() => {
    createAceSpy();

    jest
      .spyOn(hostActions, "loadAll")
      .mockImplementation(() => () => Promise.resolve([]));
  });

  const store = {
    app: {
      isSmallNav: false,
      config: configStub,
    },
    components: {
      QueryPages: {
        queryText: "SELECT * FROM users",
        selectedOsqueryTable: defaultSelectedOsqueryTable,
        selectedTargets: [],
      },
    },
    entities: {
      hosts: {
        data: {
          [hostStub.id]: hostStub,
          99: { ...hostStub, id: 99 },
        },
      },
      queries: { loading: false, data: {} },
      targets: {},
    },
    // THIS WAS ADDED 5/24, not sure if it's correct
    auth: {
      user: {
        ...adminUserStub,
      },
    },
  };
  const mockStore = reduxMockStore(store);

  describe("rendering", () => {
    it("does not render when queries are loading", () => {
      const loadingQueriesStore = {
        ...store,
        entities: {
          ...store.entities,
          queries: { loading: true, data: {} },
        },
      };
      const page = mount(
        connectedComponent(ConnectedQueryPage, {
          mockStore: reduxMockStore(loadingQueriesStore),
          props: locationProp,
        })
      );

      expect(page.html()).toBeFalsy();
    });

    it("renders the QueryForm component", () => {
      const page = mount(
        connectedComponent(ConnectedQueryPage, {
          mockStore,
          props: locationProp,
        })
      );

      expect(page.find("QueryForm").length).toEqual(1);
    });

    it("renders the QuerySidePanel component", () => {
      const page = mount(
        connectedComponent(ConnectedQueryPage, {
          mockStore,
          props: locationProp,
        })
      );

      expect(page.find("QuerySidePanel").length).toEqual(1);
    });
  });

  it("sets selectedTargets in redux based on host_ids", () => {
    const singleHostMockStore = reduxMockStore(store);
    const multipleHostMockStore = reduxMockStore(store);
    const singleHostProps = {
      params: {},
      location: { query: { host_ids: String(hostStub.id) } },
    };
    const multipleHostsProps = {
      params: {},
      location: { query: { host_ids: [String(hostStub.id), "99"] } },
    };

    mount(
      connectedComponent(ConnectedQueryPage, {
        mockStore: singleHostMockStore,
        props: singleHostProps,
      })
    );

    mount(
      connectedComponent(ConnectedQueryPage, {
        mockStore: multipleHostMockStore,
        props: multipleHostsProps,
      })
    );

    expect(singleHostMockStore.getActions()).toContainEqual({
      type: "SET_SELECTED_TARGETS",
      payload: {
        selectedTargets: [hostStub],
      },
    });

    expect(multipleHostMockStore.getActions()).toContainEqual({
      type: "SET_SELECTED_TARGETS",
      payload: {
        selectedTargets: [hostStub, { ...hostStub, id: 99 }],
      },
    });
  });

  it("sets targetError in state when the query is run and there are no selected targets", () => {
    const page = mount(
      connectedComponent(ConnectedQueryPage, { mockStore, props: locationProp })
    );
    const runQueryBtn = page.find(".query-progress-details__run-btn");
    let QueryPageSelectTargets = page.find("QueryPageSelectTargets");

    expect(QueryPageSelectTargets.prop("error")).toBeFalsy();

    runQueryBtn.hostNodes().simulate("click");

    QueryPageSelectTargets = page.find("QueryPageSelectTargets");
    expect(QueryPageSelectTargets.prop("error")).toEqual(
      "You must select at least one target to run a query"
    );
  });

  it("sets targetError in state when the query is run and the selected target contains no hosts", () => {
    const selectedTargetStore = {
      ...store,
      components: {
        ...store.components,
        QueryPages: {
          ...store.components.QueryPages,
          selectedTargets: [{ ...labelStub, count: 0 }],
        },
      },
    };
    const page = mount(
      connectedComponent(ConnectedQueryPage, {
        mockStore: reduxMockStore(selectedTargetStore),
        props: locationProp,
      })
    );
    const runQueryBtn = page.find(".query-progress-details__run-btn");
    let QueryPageSelectTargets = page.find("QueryPageSelectTargets");

    expect(QueryPageSelectTargets.prop("error")).toBeFalsy();

    runQueryBtn.hostNodes().simulate("click");

    QueryPageSelectTargets = page.find("QueryPageSelectTargets");
    expect(QueryPageSelectTargets.prop("error")).toEqual(
      "You must select a target with at least one host to run a query"
    );
  });

  it("calls the onUpdateQuery prop when the query is updated", () => {
    const query = {
      id: 1,
      name: "My query",
      description: "My query description",
      query: "select * from users",
    };
    const locationWithQueryProp = {
      params: { id: 1 },
      location: { query: {} },
    };
    const mockStoreWithQuery = reduxMockStore({
      app: {
        isSmallNav: false,
        config: configStub,
      },
      components: {
        QueryPages: {
          queryText: "SELECT * FROM users",
          selectedOsqueryTable: defaultSelectedOsqueryTable,
          selectedTargets: [],
        },
      },
      entities: {
        queries: {
          data: {
            1: query,
          },
        },
      },
      auth: {
        user: adminUserStub,
      },
    });
    const page = mount(
      connectedComponent(ConnectedQueryPage, {
        mockStore: mockStoreWithQuery,
        props: locationWithQueryProp,
      })
    );
    const form = page.find("QueryForm");
    const nameInput = form.find({ name: "name" }).find("input");
    const saveChangesBtn = form
      .find("li.dropdown-button__option")
      .first()
      .find("Button");
    fillInFormInput(nameInput, "new name");
    jest.spyOn(queryActions, "update").mockImplementation(() => () =>
      Promise.resolve({
        description: query.description,
        name: "new name",
        queryText: "SELECT * FROM users",
      })
    );

    form.simulate("submit");
    saveChangesBtn.simulate("click");

    expect(queryActions.update).toHaveBeenCalledWith(query, {
      name: "new name",
    });
  });

  describe("#componentWillReceiveProps", () => {
    it("resets selected targets and removed the campaign when the hostname changes", () => {
      const queryResult = {
        org_name: "example",
        org_url: "https://example.com",
      };
      const campaign = {
        id: 1,
        query_results: [queryResult],
        hosts_count: { total: 1 },
        Metrics: { OnlineHosts: 1, OfflineHosts: 0 },
      };
      const props = {
        dispatch: noop,
        loadingQueries: false,
        location: { pathname: "/queries/11" },
        query: { query: "select * from users" },
        selectedOsqueryTable: defaultSelectedOsqueryTable,
        selectedTargets: [hostStub],
        currentUser: adminUserStub,
      };
      const Page = mount(<QueryPage {...props} />);
      const PageNode = Page.instance();

      jest.spyOn(PageNode, "destroyCampaign");
      jest.spyOn(PageNode, "removeSocket");
      jest.spyOn(queryPageActions, "setSelectedTargets");

      Page.setState({ campaign });
      Page.setProps({ location: { pathname: "/queries/new" } });

      expect(queryPageActions.setSelectedTargets).toHaveBeenCalledWith([]);
      expect(PageNode.destroyCampaign).toHaveBeenCalled();
      expect(PageNode.removeSocket).toHaveBeenCalled();
    });
  });

  describe("export as csv", () => {
    it("exports the campaign query results in csv format", () => {
      const queryResult = {
        org_name: "example",
        org_url: "https://example.com",
      };
      const campaign = {
        id: 1,
        hosts_count: {
          failed: 0,
          successful: 1,
          total: 1,
        },
        Metrics: {
          OnlineHosts: 1,
          OfflineHosts: 0,
        },
        query_results: [queryResult],
      };
      const queryResultsCSV = convertToCSV([queryResult]);
      const fileSaveSpy = jest.spyOn(FileSave, "saveAs");
      const Page = mount(
        <QueryPage
          dispatch={noop}
          query={queryStub}
          selectedOsqueryTable={defaultSelectedOsqueryTable}
          currentUser={adminUserStub}
        />
      );
      const filename = "query_results.csv";
      const fileStub = new global.window.File([queryResultsCSV], filename, {
        type: "text/csv",
      });

      Page.setState({ campaign });
      Page.instance().socket = {};

      const QueryResultsTable = Page.find("QueryResultsTable");

      QueryResultsTable.find(".query-results-table__export-btn")
        .hostNodes()
        .simulate("click");

      expect(fileSaveSpy).toHaveBeenCalledWith(fileStub);
    });
  });

  describe("toggle full screen results", () => {
    it("toggles query results table from default to full screen and back", () => {
      const queryResult = {
        org_name: "example",
        org_url: "https://example.com",
      };
      const campaign = {
        id: 1,
        hosts_count: {
          failed: 0,
          successful: 1,
          total: 1,
        },
        Metrics: {
          OnlineHosts: 1,
          OfflineHosts: 0,
        },
        query_results: [queryResult],
      };
      const Page = mount(
        <QueryPage
          dispatch={noop}
          query={queryStub}
          selectedOsqueryTable={defaultSelectedOsqueryTable}
          currentUser={adminUserStub}
        />
      );
      Page.setState({ campaign });

      let QueryResultsTable = Page.find("QueryResultsTable");

      QueryResultsTable.find(".query-results-table__fullscreen-btn")
        .hostNodes()
        .simulate("click");

      QueryResultsTable = Page.find("QueryResultsTable");
      expect(
        QueryResultsTable.find(".query-results-table__fullscreen-btn--active")
          .length
      ).toBeGreaterThan(0);
      expect(
        QueryResultsTable.find(".query-results-table--full-screen").length
      ).toEqual(1);
      expect(Page.find(".query-page__results--full-screen").length).toEqual(1);

      QueryResultsTable.find(".query-results-table__fullscreen-btn")
        .hostNodes()
        .simulate("click");

      QueryResultsTable = Page.find("QueryResultsTable");
      expect(
        QueryResultsTable.find(".query-results-table__fullscreen-btn--active")
          .length
      ).toEqual(0);
      expect(
        QueryResultsTable.find(".query-results-table--full-screen").length
      ).toEqual(0);
      expect(
        QueryResultsTable.find(".query-results-table--shrinking").length
      ).toEqual(1);
      expect(Page.find(".query-page__results--full-screen").length).toEqual(0);
    });
  });
});
