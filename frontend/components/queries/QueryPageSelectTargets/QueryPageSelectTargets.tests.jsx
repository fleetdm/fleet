import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import QueryPageSelectTargets from "components/queries/QueryPageSelectTargets";

describe("QueryPageSelectTargets - component", () => {
  const DEFAULT_CAMPAIGN = {
    hosts_count: {
      total: 0,
    },
  };

  const defaultProps = {
    campaign: DEFAULT_CAMPAIGN,
    onFetchTargets: noop,
    onRunQuery: noop,
    onStopQuery: noop,
    onTargetSelect: noop,
    query: "select * from users",
    queryIsRunning: false,
    selectedTargets: [],
    targetsCount: 0,
  };

  describe("rendering", () => {
    const DefaultComponent = mount(
      <QueryPageSelectTargets {...defaultProps} />
    );

    it("renders", () => {
      expect(DefaultComponent.length).toEqual(
        1,
        "QueryPageSelectTargets did not render"
      );
    });

    it("renders a SelectTargetsDropdown component", () => {
      const SelectTargetsDropdown = DefaultComponent.find(
        "SelectTargetsDropdown"
      );

      expect(SelectTargetsDropdown.length).toEqual(
        1,
        "SelectTargetsDropdown did not render"
      );
    });

    it("renders a QueryProgressDetails component", () => {
      const QueryProgressDetails = DefaultComponent.find(
        "QueryProgressDetails"
      );

      expect(QueryProgressDetails.length).toEqual(
        1,
        "QueryProgressDetails did not render"
      );
    });
  });
});
