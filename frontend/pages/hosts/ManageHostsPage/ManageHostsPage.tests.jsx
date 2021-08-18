import React from "react";
import { mount, shallow } from "enzyme";
import { noop } from "lodash";

import hostActions from "redux/nodes/entities/hosts/actions";
import labelActions from "redux/nodes/entities/labels/actions";
import ConnectedManageHostsPage, {
  ManageHostsPage,
} from "pages/hosts/ManageHostsPage/ManageHostsPage";
import {
  connectedComponent,
  createAceSpy,
  reduxMockStore,
  stubbedOsqueryTable,
} from "test/helpers";
import { hostStub, configStub, adminUserStub, teamStub } from "test/stubs";
import * as manageHostsPageActions from "redux/nodes/components/ManageHostsPage/actions";

const allHostsLabel = {
  id: 1,
  display_text: "All Hosts",
  slug: "all-hosts",
  type: "all",
  count: 22,
};
const windowsLabel = {
  id: 2,
  display_text: "Windows",
  slug: "windows",
  type: "platform",
  count: 22,
};
const offlineHost = { ...hostStub, id: 111, status: "offline" };
const offlineHostsLabel = {
  id: 5,
  display_text: "OFFLINE",
  slug: "offline",
  status: "offline",
  type: "status",
  count: 1,
};
const customLabel = {
  id: 12,
  display_text: "Custom Label",
  slug: "labels/12",
  type: "custom",
  count: 3,
};
const mockStore = reduxMockStore({
  app: { enrollSecret: [], config: {} },
  auth: { user: adminUserStub },
  components: {
    ManageHostsPage: {
      display: "Grid",
      selectedLabel: {
        id: 100,
        display_text: "All Hosts",
        type: "all",
        count: 22,
      },
      status_labels: {},
    },
    QueryPages: {
      selectedOsqueryTable: stubbedOsqueryTable,
    },
  },
  entities: {
    hosts: {
      data: {
        [hostStub.id]: hostStub,
        [offlineHost.id]: offlineHost,
      },
      originalOrder: [hostStub.id, offlineHost.id],
    },
    labels: {
      data: {
        1: allHostsLabel,
        2: windowsLabel,
        3: {
          id: 3,
          display_text: "Ubuntu",
          slug: "ubuntu",
          type: "platform",
          count: 22,
        },
        4: {
          id: 4,
          display_text: "ONLINE",
          slug: "online",
          type: "status",
          count: 22,
        },
        5: offlineHostsLabel,
        6: customLabel,
      },
    },
    teams: {
      data: { [teamStub.id]: teamStub },
    },
  },
});

describe("ManageHostsPage - component", () => {
  const props = {
    config: configStub,
    dispatch: noop,
    hosts: [],
    labels: [],
    loadingHosts: false,
    loadingLabels: false,
    selectedOsqueryTable: stubbedOsqueryTable,
    statusLabels: {},
    enrollSecret: [],
  };

  beforeEach(() => {
    const spyResponse = () => Promise.resolve([]);

    jest.spyOn(hostActions, "loadAll").mockImplementation(() => spyResponse);
    jest.spyOn(labelActions, "loadAll").mockImplementation(() => spyResponse);
    jest
      .spyOn(manageHostsPageActions, "getStatusLabelCounts")
      .mockImplementation(() => spyResponse);
    createAceSpy();
  });

  describe("side panels", () => {
    it("renders a HostSidePanel when not adding a new label", () => {
      const pageProps = {
        ...props,
        selectedFilters: [],
      };

      const page = shallow(<ManageHostsPage {...pageProps} />);

      expect(page.find("HostSidePanel").length).toEqual(1);
    });

    it("renders a QuerySidePanel when adding a new label", () => {
      const ownProps = { location: { hash: "#new_label" }, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);

      expect(page.find("QuerySidePanel").length).toEqual(1);
    });
  });

  describe("Adding a new label", () => {
    beforeEach(() => createAceSpy());

    const ownProps = { location: { hash: "#new_label" }, params: {} };
    const component = connectedComponent(ConnectedManageHostsPage, {
      props: ownProps,
      mockStore,
    });

    it("renders a LabelForm component", () => {
      const page = mount(component);

      expect(page.find("LabelForm").length).toEqual(1);
    });

    it('displays "New label" as the query form header', () => {
      const page = mount(component);

      expect(page.find("LabelForm").text()).toContain("New label");
    });
  });

  describe("Active label", () => {
    beforeEach(() => createAceSpy());

    it("Displays the all hosts label as the active label by default", () => {
      const ownProps = { location: {}, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);

      expect(page.find("HostSidePanel").props()).toMatchObject({
        selectedFilter: "all-hosts",
      });
    });

    it("Displays the windows label as the active label", () => {
      const ownProps = { location: {}, params: { active_label: "labels/4" } };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);

      expect(page.find("HostSidePanel").props()).toMatchObject({
        selectedFilter: "labels/4",
      });
    });

    it("Renders the label description if the selected label has a description", () => {
      const labelDescription = "This is the label description";
      const descriptionLabel = {
        ...customLabel,
        description: labelDescription,
      };
      const pageProps = {
        ...props,
        selectedLabel: descriptionLabel,
        selectedFilters: [],
      };

      const Page = shallow(<ManageHostsPage {...pageProps} />);

      expect(
        Page.find(".manage-hosts__label-block .description span").text()
      ).toContain(labelDescription);
    });
  });

  describe("Edit a label", () => {
    const ownProps = {
      location: { hash: "" },
      params: { label_id: "12" },
    };
    const component = connectedComponent(ConnectedManageHostsPage, {
      props: ownProps,
      mockStore,
    });

    it("renders the Edit button when a custom label is selected", () => {
      const Page = mount(component);
      const EditButton = Page.find(".manage-hosts__label-block")
        .find("Button")
        .first();

      expect(EditButton.length).toEqual(1);
    });

    const ownPropsEditSelected = {
      location: { hash: "#edit_label" },
      params: { label_id: "12" },
    };
    const componentWithEditSelected = connectedComponent(
      ConnectedManageHostsPage,
      {
        props: ownPropsEditSelected,
        mockStore,
      }
    );

    it("renders a LabelForm component", () => {
      const page = mount(componentWithEditSelected);

      expect(page.find("LabelForm").length).toEqual(1);
    });

    it('displays "Edit label" as the query form header', () => {
      const page = mount(componentWithEditSelected);

      expect(page.find("LabelForm").text()).toContain("Edit label");
    });
  });

  describe("Delete a label", () => {
    it("Deleted label after confirmation modal", () => {
      const ownProps = {
        location: {},
        params: { label_id: "12" },
      };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);
      const deleteBtn = page
        .find(".manage-hosts__label-block")
        .find("Button")
        .last();

      jest
        .spyOn(labelActions, "destroy")
        .mockImplementation(() => (dispatch) => {
          dispatch({ type: "labels_LOAD_REQUEST" });

          return Promise.resolve();
        });

      expect(page.find("Modal").length).toEqual(0);

      deleteBtn.simulate("click");

      const confirmModal = page.find("Modal");

      expect(confirmModal.length).toEqual(1);

      const confirmBtn = confirmModal.find(".button--alert");
      confirmBtn.simulate("click");

      expect(labelActions.destroy).toHaveBeenCalledWith(customLabel);
    });
  });

  describe("Add Host", () => {
    it("Open the Add Host modal", () => {
      const ownProps = { location: { hash: "" }, params: {} };
      const component = connectedComponent(ConnectedManageHostsPage, {
        props: ownProps,
        mockStore,
      });
      const page = mount(component);
      const addNewHost = page.find(".manage-hosts__add-hosts");
      addNewHost.hostNodes().simulate("click");

      expect(page.find("AddHostModal").length).toBeGreaterThan(0);
    });
  });
});
