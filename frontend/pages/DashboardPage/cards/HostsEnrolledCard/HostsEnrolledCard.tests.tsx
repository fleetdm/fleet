/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { render, screen } from "@testing-library/react";
import { InjectedRouter } from "react-router";

import HostsEnrolledCard, { formatPercent } from "./HostsEnrolledCard";

// recharts' ResponsiveContainer needs a non-zero size to render its SVG in
// jsdom. Mirror CheckerboardViz's approach: stub ResizeObserver and
// getBoundingClientRect so the chart gets real dimensions on mount.
const MOCK_WIDTH = 600;
const MOCK_HEIGHT = 250;

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
          contentRect: {
            width: MOCK_WIDTH,
            height: MOCK_HEIGHT,
          } as DOMRectReadOnly,
          borderBoxSize: [],
          contentBoxSize: [],
          devicePixelContentBoxSize: [],
        },
      ],
      (this as unknown) as ResizeObserver
    );
  }

  unobserve() {}

  disconnect() {}
}

const noopRouter = ({ push: () => undefined } as unknown) as InjectedRouter;

const counts = {
  darwin: 21925,
  windows: 120,
  linux: 18,
  chrome: 7,
  ios: 0,
  ipados: 0,
  android: 0,
};

describe("formatPercent", () => {
  it("rounds a platform's share to one decimal place", () => {
    expect(formatPercent(21925, 99.16)).toBe("99.2%");
  });

  it("shows <0.1% for a nonzero share that rounds to zero", () => {
    // e.g. a handful of hosts in a fleet of tens of thousands would otherwise
    // read as a misleading "0.0%".
    expect(formatPercent(5, 0.02)).toBe("<0.1%");
  });

  it("shows 0.0% only when the count is actually zero", () => {
    expect(formatPercent(0, 0)).toBe("0.0%");
  });
});

describe("HostsEnrolledCard", () => {
  const origGetBCR = Element.prototype.getBoundingClientRect;
  const origResizeObserver = global.ResizeObserver;

  beforeAll(() => {
    global.ResizeObserver = (MockResizeObserver as unknown) as typeof ResizeObserver;
    Element.prototype.getBoundingClientRect = function mockBCR() {
      return {
        width: MOCK_WIDTH,
        height: MOCK_HEIGHT,
        top: 0,
        left: 0,
        bottom: MOCK_HEIGHT,
        right: MOCK_WIDTH,
        x: 0,
        y: 0,
        toJSON: () => {},
      } as DOMRect;
    };
  });

  afterAll(() => {
    Element.prototype.getBoundingClientRect = origGetBCR;
    global.ResizeObserver = origResizeObserver;
  });

  it("renders the title and a labeled row for every platform", () => {
    render(
      <HostsEnrolledCard
        counts={counts}
        totalHostCount={22070}
        router={noopRouter}
      />
    );

    expect(screen.getByText("Hosts enrolled")).toBeInTheDocument();
    [
      "macOS",
      "Windows",
      "Linux",
      "ChromeOS",
      "iOS",
      "iPadOS",
      "Android",
    ].forEach((label) => {
      expect(screen.getByText(label)).toBeInTheDocument();
    });
  });
});
