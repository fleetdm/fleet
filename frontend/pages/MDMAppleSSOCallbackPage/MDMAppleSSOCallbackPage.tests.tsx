import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import type { Location as HistoryLocation } from "history";

import MDMAppleSSOCallbackPage from "./MDMAppleSSOCallbackPage";

const render = createCustomRenderer();

interface ICallbackQuery {
  eula_token?: string;
  profile_token?: string;
  enrollment_reference?: string;
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

  it("still confirms a successful setup experience sign-in", () => {
    renderWithQuery({ initiator: "setup_experience" });

    expect(screen.getByText(/You.{0,3}re done/)).toBeInTheDocument();
  });
});
