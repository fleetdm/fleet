import { screen } from "@testing-library/react";
import type { Location as HistoryLocation } from "history";
import React from "react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";

import MDMAppleSSOCallbackPage from "./MDMAppleSSOCallbackPage";

const render = createCustomRenderer();

interface ICallbackQuery {
  eula_token?: string;
  profile_token?: string;
  initiator?: string;
  error?: boolean;
  reason?: string;
}

describe("MDMAppleSSOCallbackPage", () => {
  const createMockLocation = (
    query: ICallbackQuery
  ): HistoryLocation<ICallbackQuery> => ({
    action: "PUSH",
    hash: "",
    key: "test-key",
    pathname: "/mdm/sso/callback",
    search: "",
    state: null,
    query,
  });

  const renderWithQuery = (query: ICallbackQuery) =>
    render(
      <MDMAppleSSOCallbackPage
        location={createMockLocation(query)}
        params={{}}
        router={createMockRouter()}
        routes={[]}
      />
    );

  it("explains the timeout when the SSO session expired", () => {
    renderWithQuery({ error: true, reason: "session_expired" });

    expect(
      screen.getByText(/Your session may have timed out/)
    ).toBeInTheDocument();
  });

  it("keeps the generic message for other failures", () => {
    renderWithQuery({ error: true });

    expect(screen.getByText(/If this keeps happening/)).toBeInTheDocument();
    expect(screen.queryByText(/timed out/)).not.toBeInTheDocument();
  });

  it("confirms a one-shot setup experience sign-in", () => {
    sessionStorage.removeItem("setup_only");
    renderWithQuery({ initiator: "setup_experience" });

    expect(screen.getByText(/You.{0,3}re done/)).toBeInTheDocument();
  });

  it("asks the user to wait after a zero-touch kiosk sign-in", () => {
    sessionStorage.setItem("setup_only", "1");
    renderWithQuery({ initiator: "setup_experience" });
    sessionStorage.removeItem("setup_only");

    expect(screen.getByText("Setting up your device")).toBeInTheDocument();
    expect(screen.getByText(/Please wait/)).toBeInTheDocument();
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });
});
