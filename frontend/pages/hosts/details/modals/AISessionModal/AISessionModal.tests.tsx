import { screen, waitFor } from "@testing-library/react";
import { format } from "date-fns";
import { noop } from "lodash";
import { delay, http, HttpResponse } from "msw";
import React from "react";

import {
  createMockGetHostAISessionEventsResponse,
  createMockHostAISession,
  createMockHostAISessionEvent,
} from "__mocks__/aiSessionsMock";
import { IHostAISession } from "interfaces/ai_sessions";
import { IGetHostAISessionEventsResponse } from "services/entities/hosts";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import AISessionModal from "./AISessionModal";

const eventsHandler = (
  response: IGetHostAISessionEventsResponse = createMockGetHostAISessionEventsResponse(),
  onRequest?: (url: URL) => void
) =>
  http.get(baseUrl("/hosts/1/ai_sessions/1/events"), ({ request }) => {
    onRequest?.(new URL(request.url));
    return HttpResponse.json(response);
  });

const renderModal = (session: IHostAISession = createMockHostAISession()) =>
  createCustomRenderer({ withBackendMock: true })(
    <AISessionModal hostId={1} session={session} onExit={noop} />
  );

describe("AISessionModal", () => {
  it("shows the session's details", async () => {
    mockServer.use(eventsHandler());
    renderModal();

    expect(
      screen.getByText("0b6f3a51-4f3e-4f39-9d4c-2f1d2a9a6c11")
    ).toBeInTheDocument();
    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("1 hr 5 mins")).toBeInTheDocument();
    expect(screen.getByText("alice@example.com")).toBeInTheDocument();
    expect(screen.getByText("claude-opus-4-1")).toBeInTheDocument();
    // Local time, so the expected text depends on the machine's timezone.
    expect(
      screen.getByText(
        format(new Date("2026-10-20T10:00:00Z"), "MMM d, yyyy, pp")
      )
    ).toBeInTheDocument();
    expect(await screen.findByText("npm test")).toBeInTheDocument();
    expect(screen.getByText("1 event")).toBeInTheDocument();
    expect(
      screen.getByText(format(new Date("2026-10-20T10:01:00Z"), "pp"))
    ).toBeInTheDocument();
  });

  it("shows placeholders for an agent that records no start or model", async () => {
    mockServer.use(eventsHandler());
    renderModal(
      createMockHostAISession({
        agent: "cursor",
        started_at: null,
        model: "",
      })
    );

    // started, duration, and model
    expect(screen.getAllByText("---")).toHaveLength(3);
    expect(await screen.findByText("npm test")).toBeInTheDocument();
  });

  it("expands an event to show its payload", async () => {
    mockServer.use(eventsHandler());
    const { user } = renderModal();

    await user.click(await screen.findByText("npm test"));

    expect(screen.getByText(/toolu_01/)).toBeInTheDocument();

    await user.click(screen.getByText("npm test"));

    expect(screen.queryByText(/toolu_01/)).not.toBeInTheDocument();
  });

  it("shows a placeholder for an event with no time", async () => {
    mockServer.use(
      eventsHandler(
        createMockGetHostAISessionEventsResponse({
          events: [createMockHostAISessionEvent({ executed_at: null })],
        })
      )
    );
    renderModal(createMockHostAISession({ model: "" }));

    await screen.findByText("npm test");
    // model and the event's time
    expect(screen.getAllByText("---")).toHaveLength(2);
  });

  it("requests the next page of events", async () => {
    const pages: (string | null)[] = [];
    mockServer.use(
      eventsHandler(
        createMockGetHostAISessionEventsResponse({
          count: 30,
          meta: { has_next_results: true, has_previous_results: false },
        }),
        (url) => pages.push(url.searchParams.get("page"))
      )
    );
    const { user } = renderModal();

    await screen.findByText("npm test");
    await user.click(screen.getByRole("button", { name: /next/i }));

    await waitFor(() => expect(pages).toEqual(["0", "1"]));
  });

  it("disables Next while the next page of events loads", async () => {
    mockServer.use(
      http.get(
        baseUrl("/hosts/1/ai_sessions/1/events"),
        async ({ request }) => {
          if (new URL(request.url).searchParams.get("page") === "1") {
            await delay("infinite");
          }
          return HttpResponse.json(
            createMockGetHostAISessionEventsResponse({
              count: 30,
              meta: { has_next_results: true, has_previous_results: false },
            })
          );
        }
      )
    );
    const { user } = renderModal();

    await screen.findByText("npm test");
    await user.click(screen.getByRole("button", { name: /next/i }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /next/i })).toBeDisabled()
    );
  });

  it("shows that events are still being collected", async () => {
    mockServer.use(
      eventsHandler(
        createMockGetHostAISessionEventsResponse({ events: [], count: 0 })
      )
    );
    renderModal(createMockHostAISession({ backfill: true }));

    expect(await screen.findByText("Collecting events")).toBeInTheDocument();
    expect(screen.queryByText("No events")).not.toBeInTheDocument();
  });

  it("notes that some events are missing while they're collected", async () => {
    mockServer.use(eventsHandler());
    renderModal(createMockHostAISession({ backfill: true }));

    await screen.findByText("npm test");
    expect(
      screen.getByText(/still collecting earlier events/)
    ).toBeInTheDocument();
  });

  it("marks a truncated payload", async () => {
    mockServer.use(
      eventsHandler(
        createMockGetHostAISessionEventsResponse({
          events: [createMockHostAISessionEvent({ truncated: true })],
        })
      )
    );
    const { user } = renderModal();

    await user.click(await screen.findByText("npm test"));

    expect(screen.getByText("Truncated to 8 KB")).toBeInTheDocument();
  });

  it("shows an empty state when the session has no events", async () => {
    mockServer.use(
      eventsHandler(
        createMockGetHostAISessionEventsResponse({ events: [], count: 0 })
      )
    );
    renderModal();

    expect(await screen.findByText("No events")).toBeInTheDocument();
  });

  it("shows an error state when events fail to load", async () => {
    // e.g. the session was purged while the modal was open
    mockServer.use(
      http.get(baseUrl("/hosts/1/ai_sessions/1/events"), () =>
        HttpResponse.json({}, { status: 404 })
      )
    );
    renderModal();

    expect(
      await screen.findByText("Something's gone wrong.")
    ).toBeInTheDocument();
  });
});
