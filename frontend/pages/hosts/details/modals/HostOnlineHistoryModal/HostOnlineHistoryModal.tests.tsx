/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";

import HostOnlineHistoryModal from "./HostOnlineHistoryModal";

const MOCK_WIDTH = 600;

// CheckerboardViz measures its container with ResizeObserver + getBoundingClientRect;
// jsdom stubs neither. Both are restored in afterAll so other test suites see the
// original globals.
class MockResizeObserver {
  callback: ResizeObserverCallback;

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }

  observe(target: Element) {
    this.callback(
      [
        {
          target,
          contentRect: { width: MOCK_WIDTH, height: 400 } as DOMRectReadOnly,
          borderBoxSize: [],
          contentBoxSize: [],
          devicePixelContentBoxSize: [],
        },
      ],
      this
    );
  }

  unobserve() {}

  disconnect() {}
}

const generateMockChartResponse = (days: number) => {
  const data = [];
  for (let d = 0; d < days; d += 1) {
    const dateStr = `2026-03-${String(d + 1).padStart(2, "0")}`;
    for (let h = 0; h < 24; h += 3) {
      data.push({
        timestamp: `${dateStr}T${String(h).padStart(2, "0")}:00:00`,
        // Single-host filter: each bucket is 0 or 1.
        value: (d + h) % 2,
      });
    }
  }
  return {
    metric: "uptime",
    visualization: "checkerboard",
    total_hosts: 1,
    resolution: "3h",
    days,
    filters: { include_host_ids: [42] },
    data,
  };
};

const chartHandler = http.get(baseUrl("/charts/uptime"), () => {
  return HttpResponse.json(generateMockChartResponse(31));
});

const emptyChartHandler = http.get(baseUrl("/charts/uptime"), () => {
  return HttpResponse.json({
    metric: "uptime",
    visualization: "checkerboard",
    total_hosts: 1,
    resolution: "3h",
    days: 31,
    filters: {},
    data: [],
  });
});

// 4xx so React Query doesn't retry — DEFAULT_USE_QUERY_OPTIONS retries 5xx.
const errorChartHandler = http.get(baseUrl("/charts/uptime"), () => {
  return HttpResponse.json({ error: "bad request" }, { status: 400 });
});

describe("HostOnlineHistoryModal", () => {
  const origGetBCR = Element.prototype.getBoundingClientRect;
  const origResizeObserver = global.ResizeObserver;

  beforeAll(() => {
    global.ResizeObserver = (MockResizeObserver as unknown) as typeof ResizeObserver;
    Element.prototype.getBoundingClientRect = function mockBCR() {
      return {
        width: MOCK_WIDTH,
        height: 400,
        top: 0,
        left: 0,
        bottom: 400,
        right: MOCK_WIDTH,
        x: 0,
        y: 0,
        toJSON: () => {},
      };
    };
  });

  afterAll(() => {
    Element.prototype.getBoundingClientRect = origGetBCR;
    global.ResizeObserver = origResizeObserver;
  });

  it("renders the checkerboard grid when data loads", async () => {
    mockServer.use(chartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    const { container } = render(
      <HostOnlineHistoryModal
        hostId={42}
        fleetId={1}
        uptimeCollectionEnabled
        onExit={jest.fn()}
      />
    );

    await waitFor(() => {
      const rects = container.querySelectorAll("rect");
      expect(rects.length).toBeGreaterThan(0);
    });
    expect(screen.getByText("Online history")).toBeInTheDocument();
  });

  it("shows the disabled state when uptime collection is off", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isGlobalAdmin: true } },
    });
    render(
      <HostOnlineHistoryModal
        hostId={42}
        fleetId={1}
        uptimeCollectionEnabled={false}
        onExit={jest.fn()}
      />
    );

    expect(
      screen.getByText(/Data collection is disabled/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Turn on/i })
    ).toBeInTheDocument();
  });

  it("shows the empty-state message when the API returns no data", async () => {
    mockServer.use(emptyChartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <HostOnlineHistoryModal
        hostId={42}
        uptimeCollectionEnabled
        onExit={jest.fn()}
      />
    );

    await screen.findByText("No chart data available yet.");
  });

  it("shows a data error when the request fails", async () => {
    mockServer.use(errorChartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <HostOnlineHistoryModal
        hostId={42}
        uptimeCollectionEnabled
        onExit={jest.fn()}
      />
    );

    await waitFor(() => {
      // DataError renders "Something's gone wrong."
      expect(screen.getByText(/Something's gone wrong/i)).toBeInTheDocument();
    });
  });

  it("calls onExit when Done is clicked", async () => {
    mockServer.use(chartHandler);
    const onExit = jest.fn();
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(
      <HostOnlineHistoryModal
        hostId={42}
        uptimeCollectionEnabled
        onExit={onExit}
      />
    );

    await user.click(await screen.findByRole("button", { name: "Done" }));
    expect(onExit).toHaveBeenCalled();
  });
});
