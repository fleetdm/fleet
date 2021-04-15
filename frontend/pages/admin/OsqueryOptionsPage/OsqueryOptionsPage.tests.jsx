import React from "react";
import { mount } from "enzyme";

import { connectedComponent, reduxMockStore } from "test/helpers";
import ConnectedOsqueryOptionsPage, {
  OsqueryOptionsPage,
} from "pages/admin/OsqueryOptionsPage/OsqueryOptionsPage";
import osqueryOptionsActions from "redux/nodes/osquery/actions";

const currentUser = {
  admin: true,
  email: "hi@gnar.dog",
  enabled: true,
  name: "Gnar Dog",
  position: "Head of Gnar",
  username: "gnardog",
};

const osqueryOptionsString =
  "spec:\n  config:\n    options:\n      logger_plugin: tls\n      pack_delimiter: /\n      logger_tls_period: 10\n      distributed_plugin: tls\n      disable_distributed: false\n      logger_tls_endpoint: /api/v1/osquery/log\n      distributed_interval: 8\n      distributed_tls_max_attempts: 5\n    decorators:\n      load:\n        - SELECT uuid AS host_uuid FROM system_info;\n        - SELECT hostname FROM system_info;\n  overrides: {}\n";

const store = {
  app: {
    config: {
      configured: true,
    },
  },
  auth: {
    user: {
      ...currentUser,
    },
  },
  osquery: {
    erros: {},
    loading: false,
    options: {},
  },
  entities: {
    users: {
      loading: false,
      data: {
        1: {
          ...currentUser,
        },
      },
    },
  },
};

describe("Osquery Options Page - Component", () => {
  beforeEach(() => {
    jest
      .spyOn(osqueryOptionsActions, "getOsqueryOptions")
      .mockImplementation(() => () => Promise.resolve([]));

    jest
      .spyOn(osqueryOptionsActions, "updateOsqueryOptions")
      .mockImplementation(() => () => Promise.resolve([]));
  });

  it("renders", () => {
    const mockStore = reduxMockStore(store);
    const page = mount(
      connectedComponent(ConnectedOsqueryOptionsPage, { mockStore })
    );

    expect(page.find("OsqueryOptionsPage").length).toEqual(1);
  });

  it("gets osquery options on mount", () => {
    const mockStore = reduxMockStore(store);

    mount(connectedComponent(ConnectedOsqueryOptionsPage, { mockStore }));

    expect(osqueryOptionsActions.getOsqueryOptions).toHaveBeenCalled();
  });

  describe("updating osquery options", () => {
    const dispatch = () => Promise.resolve();
    const props = { dispatch, options: {} };
    const pageNode = mount(<OsqueryOptionsPage {...props} />).instance();
    const updatedOptions = { osquery_options: osqueryOptionsString };

    it("updates the current osquery options with the new osquery options object", () => {
      jest.spyOn(osqueryOptionsActions, "updateOsqueryOptions");

      pageNode.onSaveOsqueryOptionsFormSubmit(updatedOptions);

      expect(osqueryOptionsActions.updateOsqueryOptions).toHaveBeenCalledWith(
        updatedOptions
      );
    });
  });
});
