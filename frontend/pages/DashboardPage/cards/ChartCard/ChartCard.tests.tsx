/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";

import ChartCard from "./ChartCard";

// Mock ResizeObserver for CheckerboardViz
const MOCK_WIDTH = 600;

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

  // eslint-disable-next-line class-methods-use-this
  unobserve() {}

  // eslint-disable-next-line class-methods-use-this
  disconnect() {}
}

const generateMockChartResponse = (metric: string, days: number) => {
  const data = [];
  for (let d = 0; d < days; d += 1) {
    const dateStr = `2026-03-${String(d + 1).padStart(2, "0")}`;
    for (let h = 0; h < 24; h += 2) {
      data.push({
        timestamp: `${dateStr}T${String(h).padStart(2, "0")}:00:00`,
        value: Math.floor(Math.random() * 100),
      });
    }
  }
  return {
    metric,
    visualization: metric === "uptime" ? "checkerboard" : "line",
    total_hosts: 100,
    resolution: "2h",
    days,
    filters: {},
    data,
  };
};

const chartHandler = http.get(baseUrl("/charts/:metric"), ({ params }) => {
  const metric = params.metric as string;
  return HttpResponse.json(generateMockChartResponse(metric, 30));
});

const emptyChartHandler = http.get(baseUrl("/charts/:metric"), () => {
  return HttpResponse.json({
    metric: "uptime",
    visualization: "checkerboard",
    total_hosts: 0,
    resolution: "2h",
    days: 30,
    filters: {},
    data: [],
  });
});

describe("ChartCard", () => {
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

  it("renders the checkerboard visualization for uptime (default)", async () => {
    mockServer.use(chartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    const { container } = render(<ChartCard />);

    // Wait for data to load — checkerboard cells should appear
    await waitFor(() => {
      const rects = container.querySelectorAll("rect");
      expect(rects.length).toBeGreaterThan(0);
    });

    // Legend should be visible
    expect(screen.getByText("No data")).toBeInTheDocument();
    expect(screen.getByText("Less")).toBeInTheDocument();
    expect(screen.getByText("More")).toBeInTheDocument();
  });

  it("shows the no-data message when API returns empty data", async () => {
    mockServer.use(emptyChartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    render(<ChartCard />);

    await screen.findByText("No chart data available yet.");
  });

  it("renders the current dataset heading", async () => {
    mockServer.use(chartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    render(<ChartCard />);

    // Only one dataset is wired up today, so it renders as a heading rather
    // than a dropdown. Days selection is fixed at 30 and has no UI yet.
    await waitFor(() => {
      expect(screen.getByText("Hosts online")).toBeInTheDocument();
    });
  });
});
