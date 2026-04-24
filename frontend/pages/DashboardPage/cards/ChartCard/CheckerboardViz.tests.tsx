/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import CheckerboardViz from "./CheckerboardViz";
import { IFormattedDataPoint } from "./types";

// Generate data points for a given number of days with 12 two-hour slots each
const generateData = (
  numDays: number,
  percentage = 50
): IFormattedDataPoint[] => {
  const points: IFormattedDataPoint[] = [];
  for (let d = 0; d < numDays; d += 1) {
    const dateStr = `2026-03-${String(d + 1).padStart(2, "0")}`;
    for (let h = 0; h < 24; h += 2) {
      const ts = `${dateStr}T${String(h).padStart(2, "0")}:00:00`;
      points.push({
        timestamp: ts,
        label: `Mar ${d + 1}, ${h}:00`,
        value: percentage,
        percentage,
      });
    }
  }
  return points;
};

// Mock ResizeObserver and getBoundingClientRect so the component
// gets a non-zero container width in jsdom
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

  unobserve() {}

  disconnect() {}
}

describe("CheckerboardViz", () => {
  const origGetBCR = Element.prototype.getBoundingClientRect;
  const origResizeObserver = global.ResizeObserver;

  beforeAll(() => {
    global.ResizeObserver = (MockResizeObserver as unknown) as typeof ResizeObserver;
    // Mock getBoundingClientRect so the initial measurement returns non-zero
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

  it("renders the correct number of cells for 3 days of data", async () => {
    const data = generateData(3);
    // selectedDays={14} → hoursPerSlot=2 → 12 hour rows per day.
    const { container } = renderWithSetup(
      <CheckerboardViz data={data} selectedDays={14} />
    );

    // 3 days × 12 hour rows = 36 cells
    await waitFor(() => {
      const rects = container.querySelectorAll("rect");
      expect(rects).toHaveLength(36);
    });
  });

  it("applies level-0 class for 0% data points", async () => {
    const data = generateData(1, 0);
    const { container } = renderWithSetup(
      <CheckerboardViz data={data} selectedDays={30} />
    );

    await waitFor(() => {
      const rects = container.querySelectorAll("rect");
      expect(rects.length).toBeGreaterThan(0);
    });

    const rects = container.querySelectorAll("rect");
    rects.forEach((rect) => {
      expect(rect.getAttribute("class")).toContain("--level-0");
    });
  });

  it("applies correct color level classes based on percentage", async () => {
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 10,
        percentage: 10,
      },
      {
        timestamp: "2026-03-01T02:00:00",
        label: "Mar 1, 2am",
        value: 30,
        percentage: 30,
      },
      {
        timestamp: "2026-03-01T04:00:00",
        label: "Mar 1, 4am",
        value: 50,
        percentage: 50,
      },
      {
        timestamp: "2026-03-01T06:00:00",
        label: "Mar 1, 6am",
        value: 70,
        percentage: 70,
      },
      {
        timestamp: "2026-03-01T08:00:00",
        label: "Mar 1, 8am",
        value: 90,
        percentage: 90,
      },
    ];

    // selectedDays={1} renders each point as its own cell without
    // slot-bucketing, so every percentage maps to a distinct color level.
    const { container } = renderWithSetup(
      <CheckerboardViz data={points} selectedDays={1} />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    const rects = container.querySelectorAll("rect");
    const classNames = Array.from(rects).map(
      (r) => r.getAttribute("class") || ""
    );
    for (let level = 1; level <= 5; level += 1) {
      expect(classNames.some((c) => c.includes(`--level-${level}`))).toBe(true);
    }
  });

  it("renders the legend with all color levels", () => {
    const data = generateData(1);
    renderWithSetup(<CheckerboardViz data={data} selectedDays={30} />);

    expect(screen.getByText("No data")).toBeInTheDocument();
    expect(screen.getByText("Less")).toBeInTheDocument();
    expect(screen.getByText("More")).toBeInTheDocument();
  });

  it("fills empty hour slots with level-0 cells", async () => {
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 80,
        percentage: 80,
      },
    ];

    // selectedDays={14} → hoursPerSlot=2 → 12 hour rows per day.
    const { container } = renderWithSetup(
      <CheckerboardViz data={points} selectedDays={14} />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    const rects = container.querySelectorAll("rect");
    const level0Count = Array.from(rects).filter((r) =>
      (r.getAttribute("class") || "").includes("--level-0")
    ).length;
    // 11 of 12 rows should be level-0 (only slot 0 has data)
    expect(level0Count).toBe(11);
  });
});
