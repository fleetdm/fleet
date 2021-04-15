import React from "react";
import { mount } from "enzyme";
import { noop } from "lodash";

import { campaignStub } from "test/stubs";
import QueryProgressDetails from "./QueryProgressDetails";

describe("QueryProgressDetails - component", () => {
  const DEFAULT_CAMPAIGN = {
    hosts_count: {
      total: 0,
    },
  };

  const defaultProps = {
    campaign: DEFAULT_CAMPAIGN,
    onRunQuery: noop,
    onStopQuery: noop,
    query: "select * from users",
    queryIsRunning: false,
  };

  describe("rendering", () => {
    const DefaultComponent = mount(<QueryProgressDetails {...defaultProps} />);

    it("renders", () => {
      expect(DefaultComponent.length).toEqual(
        1,
        "QueryProgressDetails did not render"
      );
    });

    it("renders a Run Query Button", () => {
      const RunQueryButton = DefaultComponent.find(
        ".query-progress-details__run-btn"
      );

      expect(RunQueryButton.length).toBeGreaterThan(
        0,
        "RunQueryButton did not render"
      );
    });

    it("does not render a Stop Query Button", () => {
      const StopQueryButton = DefaultComponent.find(
        ".query-progress-details__stop-btn"
      );

      expect(StopQueryButton.length).toEqual(
        0,
        "StopQueryButton is not expected to render"
      );
    });

    it("does not render a Timer component", () => {
      const Timer = DefaultComponent.find("Timer");

      expect(Timer.length).toEqual(0, "Timer is not expected to render");
    });

    it("does not render a ProgressBar component", () => {
      const ProgressBar = DefaultComponent.find("ProgressBar");

      expect(ProgressBar.length).toEqual(
        0,
        "ProgressBar is not expected to render"
      );
    });

    describe("when the campaign has results", () => {
      describe("and the query is running", () => {
        const props = {
          ...defaultProps,
          campaign: campaignStub,
          queryIsRunning: true,
        };

        const Component = mount(<QueryProgressDetails {...props} />);

        it("renders a Timer component", () => {
          const Timer = Component.find("Timer");

          expect(Timer.length).toEqual(1, "Timer is expected to render");
        });

        it("renders a Stop Query Button", () => {
          const StopQueryButton = Component.find(
            ".query-progress-details__stop-btn"
          );

          expect(StopQueryButton.length).toBeGreaterThan(
            0,
            "StopQueryButton is expected to render"
          );
        });

        it("does not render a Run Query Button", () => {
          const RunQueryButton = Component.find(
            ".query-progress-details__run-btn"
          );

          expect(RunQueryButton.length).toEqual(
            0,
            "RunQueryButton is not expected render"
          );
        });

        it("renders a ProgressBar component", () => {
          const ProgressBar = Component.find("ProgressBar");

          expect(ProgressBar.length).toEqual(
            1,
            "ProgressBar is expected to render"
          );
        });
      });

      describe("and the query is not running", () => {
        const props = {
          ...defaultProps,
          campaign: campaignStub,
          queryIsRunning: false,
        };
        const Component = mount(<QueryProgressDetails {...props} />);

        it("does not render a Timer component", () => {
          const Timer = Component.find("Timer");

          expect(Timer.length).toEqual(0, "Timer is not expected to render");
        });

        it("does not render a Stop Query Button", () => {
          const StopQueryButton = Component.find(
            ".query-progress-details__stop-btn"
          );

          expect(StopQueryButton.length).toEqual(
            0,
            "StopQueryButton is not expected to render"
          );
        });

        it("renders a Run Query Button", () => {
          const RunQueryButton = Component.find(
            ".query-progress-details__run-btn"
          );

          expect(RunQueryButton.length).toBeGreaterThan(
            0,
            "RunQueryButton did not render"
          );
        });

        it("renders a ProgressBar component", () => {
          const ProgressBar = Component.find("ProgressBar");

          expect(ProgressBar.length).toEqual(
            1,
            "ProgressBar is expected to render"
          );
        });
      });
    });
  });

  describe("when the campaign is empty", () => {
    describe("and the query is running", () => {
      const noResults = { failed: 0, successful: 0, total: 0 };
      const campaignWithNoResults = Object.assign({}, campaignStub, {
        hosts_count: noResults,
      });
      const props = {
        ...defaultProps,
        campaign: campaignWithNoResults,
        queryIsRunning: true,
      };
      const Component = mount(<QueryProgressDetails {...props} />);

      it("renders a ProgressBar component", () => {
        const ProgressBar = Component.find("ProgressBar");

        expect(ProgressBar.length).toEqual(
          1,
          "ProgressBar is expected to render"
        );
      });
    });
  });

  describe("running a query", () => {
    it("calls the onRunQuery prop with the query text", () => {
      const spy = jest.fn();
      const props = {
        ...defaultProps,
        campaign: campaignStub,
        onRunQuery: spy,
      };
      const Component = mount(<QueryProgressDetails {...props} />);
      const RunQueryButton = Component.find(".query-progress-details__run-btn");

      RunQueryButton.hostNodes().simulate("click");

      expect(spy).toHaveBeenCalled();
    });
  });

  describe("stopping a query", () => {
    it("calls the onStopQuery prop", () => {
      const spy = jest.fn();
      const props = {
        ...defaultProps,
        campaign: campaignStub,
        onStopQuery: spy,
        queryIsRunning: true,
      };
      const Component = mount(<QueryProgressDetails {...props} />);
      const StopQueryButton = Component.find(
        ".query-progress-details__stop-btn"
      );

      StopQueryButton.hostNodes().simulate("click");

      expect(spy).toHaveBeenCalled();
    });
  });
});
