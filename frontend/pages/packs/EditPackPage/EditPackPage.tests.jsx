import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { connectedComponent, reduxMockStore } from "test/helpers";
import {
  packStub,
  queryStub,
  scheduledQueryStub,
  configStub,
} from "test/stubs";
import ConnectedEditPackPage, {
  EditPackPage,
} from "pages/packs/EditPackPage/EditPackPage";
import hostActions from "redux/nodes/entities/hosts/actions";
import labelActions from "redux/nodes/entities/labels/actions";
import packActions from "redux/nodes/entities/packs/actions";
import queryActions from "redux/nodes/entities/queries/actions";
import scheduledQueryActions from "redux/nodes/entities/scheduled_queries/actions";
import teamActions from "redux/nodes/entities/teams/actions";

describe("EditPackPage - component", () => {
  beforeEach(() => {
    const spyResponse = () => Promise.resolve([]);

    jest.spyOn(hostActions, "loadAll").mockImplementation(() => spyResponse);
    jest.spyOn(labelActions, "loadAll").mockImplementation(() => spyResponse);
    jest.spyOn(packActions, "load").mockImplementation(() => spyResponse);
    jest.spyOn(queryActions, "loadAll").mockImplementation(() => spyResponse);
    jest.spyOn(teamActions, "loadAll").mockImplementation(() => spyResponse);
    jest
      .spyOn(scheduledQueryActions, "loadAll")
      .mockImplementation(() => spyResponse);
  });

  const store = {
    app: { config: configStub },
    entities: {
      hosts: { loading: false, data: {} },
      labels: { loading: false, data: {} },
      teams: { loading: false, data: {} },
      packs: {
        loading: false,
        data: {
          [packStub.id]: packStub,
        },
      },
      scheduled_queries: { loading: false, data: {} },
    },
  };
  const page = mount(
    connectedComponent(ConnectedEditPackPage, {
      props: { params: { id: String(packStub.id) }, route: {} },
      mockStore: reduxMockStore(store),
    })
  );

  describe("rendering", () => {
    it("does not render when packs are loading", () => {
      const packsLoadingStore = {
        app: store.app,
        entities: {
          ...store.entities,
          packs: { ...store.entities.packs, loading: true },
        },
      };

      const loadingPacksPage = mount(
        connectedComponent(ConnectedEditPackPage, {
          props: { params: { id: String(packStub.id) }, route: {} },
          mockStore: reduxMockStore(packsLoadingStore),
        })
      );

      expect(loadingPacksPage.html()).toBeFalsy();
    });

    it("does not render when scheduled queries are loading", () => {
      const scheduledQueriesLoadingStore = {
        app: store.app,
        entities: {
          ...store.entities,
          scheduled_queries: {
            ...store.entities.scheduled_queries,
            loading: true,
          },
        },
      };

      const loadingScheduledQueriesPage = mount(
        connectedComponent(ConnectedEditPackPage, {
          props: { params: { id: String(packStub.id) }, route: {} },
          mockStore: reduxMockStore(scheduledQueriesLoadingStore),
        })
      );

      expect(loadingScheduledQueriesPage.html()).toBeFalsy();
    });

    it("does not render when there is no pack", () => {
      const noPackStore = {
        app: store.app,
        entities: {
          ...store.entities,
          packs: { data: {}, loading: false },
        },
      };

      const noPackPage = mount(
        connectedComponent(ConnectedEditPackPage, {
          props: { params: { id: String(packStub.id) }, route: {} },
          mockStore: reduxMockStore(noPackStore),
        })
      );

      expect(noPackPage.html()).toBeFalsy();
    });

    it("renders", () => {
      expect(page.length).toEqual(1);
    });

    it("renders a EditPackFormWrapper component", () => {
      expect(page.find("EditPackFormWrapper").length).toEqual(1);
    });

    it("renders a ScheduleQuerySidePanel component", () => {
      expect(page.find("ScheduleQuerySidePanel").length).toEqual(1);
    });
  });

  describe("updating a pack", () => {
    it("only sends the updated attributes to the server", () => {
      jest.spyOn(packActions, "update");
      const dispatch = () => Promise.resolve();

      const updatedAttrs = { name: "Updated pack name" };
      const updatedPack = { ...packStub, ...updatedAttrs };
      const props = {
        allQueries: [],
        dispatch,
        isEdit: false,
        packHosts: [],
        packLabels: [],
        packTeams: [],
        scheduledQueries: [],
      };

      const pageNode = mount(
        <EditPackPage {...props} pack={packStub} />
      ).instance();

      pageNode.handlePackFormSubmit(updatedPack);

      expect(packActions.update).toHaveBeenCalledWith(packStub, updatedAttrs);
    });
  });

  describe("updating a scheduled query", () => {
    const scheduledQuery = { ...scheduledQueryStub, query_id: queryStub.id };
    const defaultProps = {
      allQueries: [queryStub],
      dispatch: noop,
      isEdit: true,
      isLoadingPack: false,
      isLoadingScheduledQueries: false,
      pack: packStub,
      packHosts: [],
      packID: String(packStub.id),
      packLabels: [],
      packTeams: [],
      scheduledQueries: [scheduledQuery],
    };

    it("de-selects the scheduledQuery when cancel is clicked", () => {
      const Form = (Page) => Page.find("ConfigurePackQueryForm");
      const Page = mount(<EditPackPage {...defaultProps} />);
      const QueryRow = Page.find("ScheduledQueriesList").find(
        "ClickableTableRow"
      );

      expect(Page.instance().state.selectedScheduledQuery).toBeFalsy();

      QueryRow.simulate("click");

      expect(Page.state().selectedScheduledQuery).toEqual(
        scheduledQuery,
        "Expected clicking a scheduled query row to set the scheduled query in component state"
      );

      const PageForm = Form(Page);

      expect(PageForm.length).toEqual(
        1,
        "Expected clicking a scheduled query row to render the ConfigurePackQueryForm component"
      );

      PageForm.find(".configure-pack-query-form__cancel-btn")
        .hostNodes()
        .simulate("click");

      expect(Page.state().selectedScheduledQuery).toBeFalsy();

      expect(Form(Page).length).toEqual(
        0,
        "Expected clicking Cancel to remove the ConfigurePackQueryForm component"
      );
    });
  });

  describe("double clicking a scheduled query", () => {
    const scheduledQuery = { ...scheduledQueryStub, query_id: queryStub.id };
    const defaultProps = {
      allQueries: [queryStub],
      dispatch: jest.fn(),
      isEdit: true,
      isLoadingPack: false,
      isLoadingScheduledQueries: false,
      pack: packStub,
      packHosts: [],
      packID: String(packStub.id),
      packLabels: [],
      packTeams: [],
      scheduledQueries: [scheduledQuery],
    };
    const pushAction = {
      type: "@@router/CALL_HISTORY_METHOD",
      payload: {
        method: "push",
        args: [`/queries/${scheduledQuery.query_id}`],
      },
    };

    it("should take user to edit query page", () => {
      const Page = mount(<EditPackPage {...defaultProps} />).find(
        "EditPackPage"
      );
      const QueryRow = Page.find("ScheduledQueriesList").find(
        "ClickableTableRow"
      );

      QueryRow.simulate("doubleclick");

      expect(defaultProps.dispatch).toHaveBeenCalledWith(pushAction);
    });
  });
});
