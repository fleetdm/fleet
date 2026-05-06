/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import { IFormattedDataPoint } from "interfaces/charts";

import CheckerboardViz from "./CheckerboardViz";

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

  it("stretches the color ramp across the dataset's min/max when relativeScale is true", async () => {
    // All values fall in the 0–20% range, which the default fixed-threshold
    // ramp would collapse entirely into level-1 (with 0% pinned to level-0).
    // Relative scaling should rescale the non-zero values across levels 1–5.
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 0,
        percentage: 0,
      },
      {
        timestamp: "2026-03-01T02:00:00",
        label: "Mar 1, 2am",
        value: 4,
        percentage: 4,
      },
      {
        timestamp: "2026-03-01T04:00:00",
        label: "Mar 1, 4am",
        value: 8,
        percentage: 8,
      },
      {
        timestamp: "2026-03-01T06:00:00",
        label: "Mar 1, 6am",
        value: 12,
        percentage: 12,
      },
      {
        timestamp: "2026-03-01T08:00:00",
        label: "Mar 1, 8am",
        value: 16,
        percentage: 16,
      },
      {
        timestamp: "2026-03-01T10:00:00",
        label: "Mar 1, 10am",
        value: 20,
        percentage: 20,
      },
    ];

    // Sanity check: with the default fixed ramp, every non-zero point lands
    // in level-1.
    const { container: defaultContainer } = renderWithSetup(
      <CheckerboardViz data={points} selectedDays={1} />
    );
    await waitFor(() => {
      expect(defaultContainer.querySelectorAll("rect").length).toBeGreaterThan(
        0
      );
    });
    const defaultClasses = Array.from(
      defaultContainer.querySelectorAll("rect")
    ).map((r) => r.getAttribute("class") || "");
    for (let level = 2; level <= 5; level += 1) {
      expect(defaultClasses.some((c) => c.includes(`--level-${level}`))).toBe(
        false
      );
    }

    const { container } = renderWithSetup(
      <CheckerboardViz data={points} selectedDays={1} relativeScale />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    const classNames = Array.from(container.querySelectorAll("rect")).map(
      (r) => r.getAttribute("class") || ""
    );
    // 0% stays reserved for level-0 even when the ramp is stretched.
    expect(classNames.some((c) => c.includes("--level-0"))).toBe(true);
    // The remaining values should span all five non-zero levels.
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

  it("applies the default green theme class when no theme prop is passed", async () => {
    const data = generateData(1);
    const { container } = renderWithSetup(
      <CheckerboardViz data={data} selectedDays={30} />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    expect(
      container.querySelector(".checkerboard-viz--theme-green")
    ).toBeInTheDocument();
    expect(
      container.querySelector(".checkerboard-viz--theme-red")
    ).not.toBeInTheDocument();
  });

  it("applies the red theme class when theme='red' is passed", async () => {
    const data = generateData(1);
    const { container } = renderWithSetup(
      <CheckerboardViz data={data} selectedDays={30} theme="red" />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    expect(
      container.querySelector(".checkerboard-viz--theme-red")
    ).toBeInTheDocument();
    expect(
      container.querySelector(".checkerboard-viz--theme-green")
    ).not.toBeInTheDocument();
  });

  it("renders the default '<percentage>% of hosts' tooltip when no formatter is provided", async () => {
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 42,
        percentage: 70,
        total: 60,
      },
    ];

    const { container } = renderWithSetup(
      <CheckerboardViz data={points} selectedDays={14} />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    // The first cell (slot 0) is the one populated by the data point above.
    fireEvent.mouseEnter(container.querySelector("rect") as SVGRectElement);

    expect(await screen.findByText("70% of hosts")).toBeInTheDocument();
  });

  it("invokes tooltipFormatter with the cell's value, percentage, and total", async () => {
    const formatter = jest.fn(({ value, total, percentage }) => (
      <span data-testid="custom-tooltip">
        {percentage}% exposed ({value} / {total} hosts)
      </span>
    ));
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 12,
        percentage: 20,
        total: 60,
      },
    ];

    const { container } = renderWithSetup(
      <CheckerboardViz
        data={points}
        selectedDays={14}
        tooltipFormatter={formatter}
      />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    fireEvent.mouseEnter(container.querySelector("rect") as SVGRectElement);

    const tooltip = await screen.findByTestId("custom-tooltip");
    expect(tooltip).toHaveTextContent("20% exposed (12 / 60 hosts)");
    expect(formatter).toHaveBeenCalledWith({
      value: 12,
      total: 60,
      percentage: 20,
    });
  });

  it("passes total=0 through to tooltipFormatter when the dataset reports zero hosts", async () => {
    const formatter = jest.fn(({ value, total, percentage }) => (
      <span data-testid="custom-tooltip">
        {percentage}% ({value} / {total} hosts)
      </span>
    ));
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 0,
        percentage: 0,
        total: 0,
      },
    ];

    const { container } = renderWithSetup(
      <CheckerboardViz
        data={points}
        selectedDays={14}
        tooltipFormatter={formatter}
      />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    fireEvent.mouseEnter(container.querySelector("rect") as SVGRectElement);

    const tooltip = await screen.findByTestId("custom-tooltip");
    expect(tooltip).toHaveTextContent("0% (0 / 0 hosts)");
    expect(formatter).toHaveBeenCalledWith({
      value: 0,
      total: 0,
      percentage: 0,
    });
  });

  it("passes total=undefined through to tooltipFormatter when the data point omits it", async () => {
    const formatter = jest.fn(({ value, total, percentage }) => (
      <span data-testid="custom-tooltip">
        {percentage}% / value={value} / total=
        {total === undefined ? "n/a" : total}
      </span>
    ));
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 5,
        percentage: 50,
        // total intentionally omitted
      },
    ];

    const { container } = renderWithSetup(
      <CheckerboardViz
        data={points}
        selectedDays={14}
        tooltipFormatter={formatter}
      />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    fireEvent.mouseEnter(container.querySelector("rect") as SVGRectElement);

    const tooltip = await screen.findByTestId("custom-tooltip");
    expect(tooltip).toHaveTextContent("50% / value=5 / total=n/a");
    expect(formatter).toHaveBeenCalledWith({
      value: 5,
      total: undefined,
      percentage: 50,
    });
  });

  it("falls back to value=0 when hovering a cell with no underlying data point", async () => {
    const formatter = jest.fn(({ value, total, percentage }) => (
      <span data-testid="custom-tooltip">
        v={value}/p={percentage}/t={total === undefined ? "n/a" : total}
      </span>
    ));
    // Single data point at slot 0; remaining 11 slots in the day are filled
    // with synthetic level-0 cells whose value/total fall back to 0/undefined.
    const points: IFormattedDataPoint[] = [
      {
        timestamp: "2026-03-01T00:00:00",
        label: "Mar 1, 12am",
        value: 80,
        percentage: 80,
        total: 100,
      },
    ];

    const { container } = renderWithSetup(
      <CheckerboardViz
        data={points}
        selectedDays={14}
        tooltipFormatter={formatter}
      />
    );

    await waitFor(() => {
      expect(container.querySelectorAll("rect").length).toBeGreaterThan(0);
    });

    const emptyCell = Array.from(container.querySelectorAll("rect")).find((r) =>
      (r.getAttribute("class") || "").includes("--level-0")
    );
    expect(emptyCell).toBeDefined();
    fireEvent.mouseEnter(emptyCell as Element);

    const tooltip = await screen.findByTestId("custom-tooltip");
    expect(tooltip).toHaveTextContent("v=0/p=0/t=n/a");
    expect(formatter).toHaveBeenCalledWith({
      value: 0,
      total: undefined,
      percentage: 0,
    });
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
