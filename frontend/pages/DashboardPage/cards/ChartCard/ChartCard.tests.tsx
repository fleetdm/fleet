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

  it("includes mobile platforms by default and does not show the Filtered badge", async () => {
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

    // No platform filter is active by default, so all platforms (including
    // iOS/iPadOS/Android) are included and the "Filtered" badge is absent.
    await waitFor(() => {
      const rects = document.querySelectorAll("rect");
      expect(rects.length).toBeGreaterThan(0);
    });
    expect(requestedPlatforms).toBeNull();
    expect(screen.queryByText("Filtered")).not.toBeInTheDocument();
  });

  describe("vulnerability exposure severity filter", () => {
    // Captures the query params of the last /charts/cve request.
    const useCveHandler = () => {
      const captured: { params: URLSearchParams | null } = { params: null };
      mockServer.use(
        http.get(baseUrl("/charts/:metric"), ({ params, request }) => {
          if (params.metric === "cve") {
            captured.params = new URL(request.url).searchParams;
          }
          return HttpResponse.json(
            generateMockChartResponse(params.metric as string, 30)
          );
        })
      );
      return captured;
    };

    const renderPremium = (
      props: React.ComponentProps<typeof ChartCard> = {}
    ) =>
      createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true } },
      })(<ChartCard {...props} />);

    // react-select renders its options as plain divs, so target them by the
    // testid the shared custom Option component sets rather than by role.
    const selectDataset = async (
      user: ReturnType<typeof renderPremium>["user"],
      label: string
    ) => {
      // Let the initial chart request settle first — the re-render it triggers
      // closes the menu again if it lands between opening and picking.
      await waitFor(() => {
        expect(document.querySelectorAll("rect").length).toBeGreaterThan(0);
      });
      await user.click(screen.getByRole("combobox", { name: "dataset" }));
      const option = screen
        .getAllByTestId("dropdown-option")
        .find((el) => el.textContent?.startsWith(label));
      if (!option) {
        throw new Error(`No dataset option matching "${label}"`);
      }
      await user.click(option);
    };

    // Which bounds a selection resolves to is severityFilters' contract. What
    // only a real request shows is the wiring either side: that they land on
    // severity_min / severity_max, and that buildQueryStringFromParams keeps an
    // explicit 0 rather than dropping it as empty.
    const boundsCases: {
      label: string;
      filterDefaults: React.ComponentProps<typeof ChartCard>["filterDefaults"];
      min: string | null;
      max: string | null;
    }[] = [
      {
        label: "the built-in critical default",
        filterDefaults: undefined,
        min: "9",
        max: "10",
      },
      {
        label: "a custom range with a 0 minimum",
        filterDefaults: { cvss_min: 0, cvss_max: 6.5 },
        min: "0",
        max: "6.5",
      },
      {
        label: "Any severity",
        filterDefaults: { cvss_min: 0, cvss_max: 10 },
        min: null,
        max: null,
      },
    ];

    it.each(boundsCases)(
      "requests $label as severity bounds",
      async ({ filterDefaults, min, max }) => {
        const captured = useCveHandler();
        const { user } = renderPremium({ filterDefaults });

        await selectDataset(user, "Vulnerability exposure");

        await waitFor(() => expect(captured.params).not.toBeNull());
        expect(captured.params?.get("severity_min")).toBe(min);
        expect(captured.params?.get("severity_max")).toBe(max);
      }
    );

    it("does not flag a seeded default as Filtered", async () => {
      useCveHandler();
      const { user } = renderPremium({
        filterDefaults: { cvss_min: 7, cvss_max: 8.9 },
      });

      await selectDataset(user, "Vulnerability exposure");

      // High severity genuinely narrows the data, but it is the chart's
      // baseline rather than something the user chose, so the pill has nothing
      // to report yet.
      expect(screen.queryByText("Filtered")).not.toBeInTheDocument();
    });

    it("lights the Filtered pill once a filter is changed off the baseline", async () => {
      useCveHandler();
      const { user } = renderPremium();

      await selectDataset(user, "Vulnerability exposure");
      expect(screen.queryByText("Filtered")).not.toBeInTheDocument();

      // Narrow the categories, which no default touched.
      await user.click(
        screen.getByRole("button", { name: /Configure chart filters/i })
      );
      await user.click(screen.getByRole("tab", { name: /Software/i }));
      await user.click(screen.getByRole("checkbox", { name: "category-os" }));
      await user.click(screen.getByRole("button", { name: /^Apply$/i }));

      // Only the pill itself: react-tooltip does not mount its content in
      // jsdom, and softwareFilterLines' tests cover the text.
      await waitFor(() => {
        expect(screen.getByText("Filtered")).toBeInTheDocument();
      });
    });

    it("opens the Advanced section when the modal is reached from the pill", async () => {
      useCveHandler();
      const { user } = renderPremium();

      await selectDataset(user, "Vulnerability exposure");

      // Light the pill by narrowing the categories, then close and reopen
      // through it.
      await user.click(
        screen.getByRole("button", { name: /Configure chart filters/i })
      );
      await user.click(screen.getByRole("tab", { name: /Software/i }));
      await user.click(screen.getByRole("checkbox", { name: "category-os" }));
      await user.click(screen.getByRole("button", { name: /^Apply$/i }));
      await waitFor(() => {
        expect(screen.getByText("Filtered")).toBeInTheDocument();
      });

      await user.click(screen.getByText("Filtered"));
      expect(screen.getByText("Probability of exploit")).toBeInTheDocument();
    });

    it("leaves the Advanced section collapsed when opened from the gear", async () => {
      useCveHandler();
      const { user } = renderPremium();

      await selectDataset(user, "Vulnerability exposure");
      await user.click(
        screen.getByRole("button", { name: /Configure chart filters/i })
      );
      await user.click(screen.getByRole("tab", { name: /Software/i }));

      expect(
        screen.queryByText("Probability of exploit")
      ).not.toBeInTheDocument();
    });

    it("no longer advertises the severity filter as coming soon", async () => {
      useCveHandler();
      const { user } = renderPremium();

      await selectDataset(user, "Vulnerability exposure");

      expect(screen.queryByText(/coming soon/i)).not.toBeInTheDocument();
      expect(
        screen.queryByText(/All critical vulnerabilities/i)
      ).not.toBeInTheDocument();
    });
  });
});
