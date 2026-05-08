import React from "react";
import { render, screen } from "@testing-library/react";

import NoResults from "./NoResults";

const baseProps = {
  queryId: 42,
  disabledCaching: false,
  disabledCachingGlobally: false,
  discardDataEnabled: false,
  loggingSnapshot: true,
};

describe("NoResults", () => {
  describe("no interval set", () => {
    it("shows interval and live report text when user can edit and run live", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={0}
          canEditQuery
          canLiveQuery
        />
      );

      expect(screen.getByText("Nothing to report")).toBeInTheDocument();
      expect(screen.getByText(/Add an/)).toBeInTheDocument();
      expect(screen.getByText("interval")).toBeInTheDocument();
      expect(screen.getByText("live report")).toBeInTheDocument();
      expect(
        screen.getByRole("link", { name: /live report/ })
      ).toHaveAttribute("href", expect.stringContaining("/reports/42/live"));
    });

    it("shows only interval text when user can edit but not run live", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={0}
          canEditQuery
          canLiveQuery={false}
        />
      );

      expect(screen.getByText(/Add an/)).toBeInTheDocument();
      expect(screen.getByText("interval")).toBeInTheDocument();
      expect(screen.queryByText("live report")).not.toBeInTheDocument();
    });

    it("shows only live report link when user can run live but not edit", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={0}
          canEditQuery={false}
          canLiveQuery
        />
      );

      expect(screen.queryByText(/Add an/)).not.toBeInTheDocument();
      expect(screen.getByText("live report")).toBeInTheDocument();
    });

    it("shows no actionable text when user can neither edit nor run live", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={0}
          canEditQuery={false}
          canLiveQuery={false}
        />
      );

      expect(
        screen.getByText(/does not collect data on a schedule/)
      ).toBeInTheDocument();
      expect(screen.queryByText(/Add an/)).not.toBeInTheDocument();
      expect(screen.queryByText("live report")).not.toBeInTheDocument();
    });
  });

  describe("has interval but no results yet", () => {
    it("shows live report link when user can run live", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={3600}
          queryUpdatedAt={new Date(0).toISOString()}
          canLiveQuery
        />
      );

      expect(screen.getByText("Nothing to report yet")).toBeInTheDocument();
      expect(screen.getByText("live report")).toBeInTheDocument();
    });

    it("does not show live report link when user cannot run live", () => {
      render(
        <NoResults
          {...baseProps}
          queryInterval={3600}
          queryUpdatedAt={new Date(0).toISOString()}
          canLiveQuery={false}
        />
      );

      expect(screen.getByText("Nothing to report yet")).toBeInTheDocument();
      expect(screen.queryByText("live report")).not.toBeInTheDocument();
    });
  });
});
