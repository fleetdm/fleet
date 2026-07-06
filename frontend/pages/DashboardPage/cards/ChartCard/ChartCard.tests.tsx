/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";

import ChartCard, {
  buildInitialChartFilters,
  hostFilterLines,
} from "./ChartCard";

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

  it("renders the empty state with a Turn on button for admins", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isGlobalAdmin: true } },
    });
    render(
      <ChartCard
        historicalDataEnabled={{ uptime: false, vulnerabilities: true }}
      />
    );

    expect(
      screen.getByText(/Data collection is disabled/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Turn on/i })
    ).toBeInTheDocument();
  });

  it("hides the Turn on button and swaps copy for non-admins", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: { app: { isGlobalAdmin: false, isTeamAdmin: false } },
    });
    render(
      <ChartCard
        historicalDataEnabled={{ uptime: false, vulnerabilities: true }}
      />
    );

    expect(
      screen.getByText(/Data collection is disabled/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Ask an admin to turn on/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Turn on/i })
    ).not.toBeInTheDocument();
  });

  it("renders the chart normally when collection is enabled", async () => {
    mockServer.use(chartHandler);
    const render = createCustomRenderer({ withBackendMock: true });
    const { container } = render(
      <ChartCard
        historicalDataEnabled={{ uptime: true, vulnerabilities: true }}
      />
    );

    await waitFor(() => {
      const rects = container.querySelectorAll("rect");
      expect(rects.length).toBeGreaterThan(0);
    });
    expect(
      screen.queryByText(/Data collection is disabled/i)
    ).not.toBeInTheDocument();
  });

  it("excludes mobile platforms by default and shows the Filtered badge", async () => {
    let requestedPlatforms: string | null = null;
    mockServer.use(
      http.get(baseUrl("/charts/:metric"), ({ params, request }) => {
        requestedPlatforms = new URL(request.url).searchParams.get("platforms");
        return HttpResponse.json(
          generateMockChartResponse(params.metric as string, 30)
        );
      })
    );
    const render = createCustomRenderer({ withBackendMock: true });
    render(<ChartCard />);

    // The default platform filter is the four non-mobile platforms, which both
    // excludes iOS/iPadOS/Android and surfaces the "Filtered" badge on load.
    await waitFor(() => {
      expect(screen.getByText("Filtered")).toBeInTheDocument();
    });
    await waitFor(() => {
      expect(requestedPlatforms).toBe("darwin,windows,linux,chrome");
    });
    expect(requestedPlatforms).not.toMatch(/ios|ipados|android/);
  });
});

describe("buildInitialChartFilters", () => {
  it("uses built-in defaults when no persisted defaults are provided", () => {
    const filters = buildInitialChartFilters(undefined);
    expect(filters.softwareFilters).toEqual([
      ...ALL_CVE_SOFTWARE_CATEGORY_VALUES,
    ]);
    expect(filters.knownExploit).toBe(false);
    expect(filters.epssMin).toBe("");
    expect(filters.epssMax).toBe("");
    expect(filters.excludeCVEs).toEqual([]);
  });

  it("seeds present fields and falls back per-field for absent ones", () => {
    const filters = buildInitialChartFilters({
      software_filters: ["browsers"],
      has_known_exploit: true,
    });
    expect(filters.softwareFilters).toEqual(["browsers"]);
    expect(filters.knownExploit).toBe(true);
    expect(filters.epssMin).toBe("");
    expect(filters.epssMax).toBe("");
    expect(filters.excludeCVEs).toEqual([]);
  });

  it("converts numeric EPSS bounds (0-100) to strings", () => {
    const filters = buildInitialChartFilters({ epss_min: 0, epss_max: 90 });
    expect(filters.epssMin).toBe("0");
    expect(filters.epssMax).toBe("90");
  });

  it("honors an explicit empty software_filters list as 'none'", () => {
    const filters = buildInitialChartFilters({ software_filters: [] });
    expect(filters.softwareFilters).toEqual([]);
  });

  it("seeds the exclude-CVE list", () => {
    const filters = buildInitialChartFilters({
      exclude_vulnerabilities: ["CVE-2025-50897"],
    });
    expect(filters.excludeCVEs).toEqual(["CVE-2025-50897"]);
  });
});

describe("hostFilterLines", () => {
  const filtersWithPlatforms = (platforms: string[]) => ({
    ...buildInitialChartFilters(undefined),
    platforms,
  });

  it("preserves branded platform casing (macOS, iOS, iPadOS)", () => {
    const [line] = hostFilterLines(
      filtersWithPlatforms(["darwin", "ios", "ipados"])
    );
    expect(line).toBe("macOS, iOS, and iPadOS");
    // Guards the reported bug: no word-capitalized variants.
    expect(line).not.toMatch(/MacOS|Ios|Ipados/);
  });

  it("renders a single platform without mangling its casing", () => {
    expect(hostFilterLines(filtersWithPlatforms(["darwin"]))).toEqual([
      "macOS",
    ]);
  });

  it("maps every filterable platform to its correct display name", () => {
    const [line] = hostFilterLines(
      filtersWithPlatforms([
        "darwin",
        "windows",
        "linux",
        "chrome",
        "ios",
        "ipados",
        "android",
      ])
    );
    expect(line).toBe(
      "macOS, Windows, Linux, ChromeOS, iOS, iPadOS, and Android"
    );
  });
});
