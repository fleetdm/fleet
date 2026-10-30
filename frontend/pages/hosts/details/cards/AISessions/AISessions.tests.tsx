import { screen, waitFor, within } from "@testing-library/react";
import { format } from "date-fns";
import { noop } from "lodash";
import React from "react";

import {
  createMockGetHostAISessionsResponse,
  createMockHostAISession,
} from "__mocks__/aiSessionsMock";
import { IGetHostAISessionsResponse } from "services/entities/hosts";
import { createCustomRenderer } from "test/test-utils";

import AISessionsCard from "./AISessions";

const baseProps = {
  page: 0,
  pageSize: 10,
  sortHeader: "started_at",
  sortDirection: "desc",
  isLoading: false,
  isPreviousData: false,
  isError: false,
  onSelectSession: noop,
  onNextPage: noop,
  onPreviousPage: noop,
  onSortChange: noop,
};

const renderCard = (
  props: Partial<typeof baseProps> = {},
  data?: IGetHostAISessionsResponse
) =>
  // SoftwareIcon, in the Agent column, queries for custom icons.
  createCustomRenderer({ withBackendMock: true })(
    <AISessionsCard
      {...baseProps}
      {...props}
      data={data ?? createMockGetHostAISessionsResponse()}
    />
  );

describe("AISessionsCard", () => {
  it("renders a session's agent, duration, user account, and events", () => {
    renderCard();

    expect(screen.getByText("AI sessions")).toBeInTheDocument();
    expect(screen.getByText("Last 30 days")).toBeInTheDocument();
    expect(screen.getByText("1 session")).toBeInTheDocument();
    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    // Local time, so the expected text depends on the machine's timezone.
    expect(
      screen.getByText(format(new Date("2026-10-20T10:00:00Z"), "MMM d, p"))
    ).toBeInTheDocument();
    expect(screen.getByText("1 hr 5 mins")).toBeInTheDocument();
    // the truncated text cell also renders its value in a tooltip
    expect(screen.getAllByText("alice@example.com")[0]).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
  });

  it("falls back to the local account when the agent records no account", () => {
    renderCard(
      {},
      createMockGetHostAISessionsResponse({
        ai_sessions: [
          createMockHostAISession({ account_identifier: "", username: "bob" }),
        ],
      })
    );

    expect(screen.getAllByText("bob")[0]).toBeInTheDocument();
  });

  it("shows placeholders for an unknown started time and duration", () => {
    renderCard(
      {},
      createMockGetHostAISessionsResponse({
        ai_sessions: [
          createMockHostAISession({ agent: "cursor", started_at: null }),
        ],
      })
    );

    const row = screen.getByText("Cursor").closest("tr") as HTMLElement;
    expect(within(row).getAllByText("---")).toHaveLength(2);
  });

  it("calls onSelectSession with the clicked session", async () => {
    const onSelectSession = jest.fn();
    const session = createMockHostAISession({ id: 42 });
    const { user } = renderCard(
      { onSelectSession },
      createMockGetHostAISessionsResponse({ ai_sessions: [session] })
    );

    await user.click(screen.getByText("Claude Code"));

    expect(onSelectSession).toHaveBeenCalledWith(session);
  });

  it("doesn't leave the clicked row focused behind the session modal", async () => {
    const { user } = renderCard();

    await user.click(screen.getByText("Claude Code"));

    expect(screen.getByText("Claude Code").closest("tr")).not.toHaveFocus();
  });

  it("sorts by the columns the API supports only", async () => {
    const onSortChange = jest.fn();
    const { user } = renderCard({ onSortChange });

    await user.click(screen.getByText("Duration"));
    await user.click(screen.getByText("Events"));
    expect(onSortChange).not.toHaveBeenCalled();

    await user.click(screen.getByText("Agent"));
    expect(onSortChange).toHaveBeenCalledWith(
      expect.objectContaining({ order_key: "agent" })
    );
  });

  it("pages through sessions", async () => {
    const onNextPage = jest.fn();
    const onPreviousPage = jest.fn();
    const { user } = renderCard(
      { page: 1, onNextPage, onPreviousPage },
      createMockGetHostAISessionsResponse({
        ai_sessions: Array.from({ length: 10 }, (_, i) =>
          createMockHostAISession({ id: i + 1 })
        ),
        count: 30,
        meta: { has_next_results: true, has_previous_results: true },
      })
    );

    await user.click(screen.getByRole("button", { name: /next/i }));
    expect(onNextPage).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /previous/i }));
    expect(onPreviousPage).toHaveBeenCalled();
  });

  it("disables the next page on the last page", () => {
    renderCard({ page: 1 });

    expect(screen.getByRole("button", { name: /next/i })).toBeDisabled();
  });

  it("goes back from a page past the last session", async () => {
    const onPreviousPage = jest.fn();
    renderCard(
      { page: 2, onPreviousPage },
      createMockGetHostAISessionsResponse({ ai_sessions: [], count: 20 })
    );

    expect(screen.queryByText("No AI sessions")).not.toBeInTheDocument();
    await waitFor(() => expect(onPreviousPage).toHaveBeenCalled());
  });

  it("doesn't open or page past sessions shown while new ones load", async () => {
    const onSelectSession = jest.fn();
    const onNextPage = jest.fn();
    const { user } = renderCard(
      { isPreviousData: true, onSelectSession, onNextPage },
      createMockGetHostAISessionsResponse({
        ai_sessions: Array.from({ length: 10 }, (_, i) =>
          createMockHostAISession({ id: i + 1 })
        ),
        count: 30,
        meta: { has_next_results: true, has_previous_results: false },
      })
    );

    await user.click(screen.getAllByText("Claude Code")[0]);

    expect(onSelectSession).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /next/i }));

    expect(onNextPage).not.toHaveBeenCalled();
  });

  it("explains an events count Fleet is still collecting", async () => {
    const { user } = renderCard(
      {},
      createMockGetHostAISessionsResponse({
        ai_sessions: [
          createMockHostAISession({ backfill: true, tool_call_count: 140 }),
        ],
      })
    );

    await user.hover(screen.getByText("140"));

    expect(
      await screen.findByText(/still collecting earlier events/)
    ).toBeInTheDocument();
  });

  it("shows a spinner while loading", async () => {
    renderCard({ isLoading: true });

    expect(await screen.findByTestId("spinner")).toBeInTheDocument();
  });

  it("shows an error state", () => {
    renderCard({ isError: true });

    expect(screen.getByText("Something's gone wrong.")).toBeInTheDocument();
  });

  it("shows an empty state when the host has no sessions", () => {
    renderCard(
      {},
      createMockGetHostAISessionsResponse({ ai_sessions: [], count: 0 })
    );

    expect(screen.getByText("No AI sessions")).toBeInTheDocument();
  });
});
