/* eslint-disable @typescript-eslint/no-empty-function, class-methods-use-this */
import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { InjectedRouter } from "react-router";
import { ILabelSummary } from "interfaces/label";

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

// Built-in labels keyed to PLATFORM_NAME_TO_LABEL_NAME so the card can resolve a
// hosts-list link for each platform.
const builtInLabels: ILabelSummary[] = [
  { id: 10, name: "macOS", label_type: "builtin" },
  { id: 11, name: "MS Windows", label_type: "builtin" },
  { id: 12, name: "All Linux", label_type: "builtin" },
  { id: 13, name: "chrome", label_type: "builtin" },
  { id: 14, name: "iOS", label_type: "builtin" },
  { id: 15, name: "iPadOS", label_type: "builtin" },
  { id: 16, name: "Android", label_type: "builtin" },
];

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

  // Regression: the per-platform labels must be keyboard-operable, not just
  // mouse-clickable SVG text. See #48214.
  describe("keyboard accessibility", () => {
    it("exposes platforms with hosts as focusable, accessibly named buttons", () => {
      render(
        <HostsEnrolledCard
          counts={counts}
          totalHostCount={22070}
          builtInLabels={builtInLabels}
          router={noopRouter}
        />
      );

      const macButton = screen.getByRole("button", { name: "macOS hosts" });
      expect(macButton).toBeInTheDocument();
      // Reachable via Tab (tabindex 0).
      expect(macButton).toHaveAttribute("tabindex", "0");

      // A platform whose display label ("ChromeOS") differs from its built-in
      // label name ("chrome") still resolves and becomes operable.
      expect(
        screen.getByRole("button", { name: "ChromeOS hosts" })
      ).toBeInTheDocument();
    });

    it("does not turn platforms with zero hosts into buttons", () => {
      render(
        <HostsEnrolledCard
          counts={counts}
          totalHostCount={22070}
          builtInLabels={builtInLabels}
          router={noopRouter}
        />
      );

      // iOS/iPadOS/Android have a count of 0, so they should stay plain text.
      expect(
        screen.queryByRole("button", { name: "iOS hosts" })
      ).not.toBeInTheDocument();
    });

    it("navigates to the platform's hosts list on Enter and Space", () => {
      const push = jest.fn();
      const router = ({ push } as unknown) as InjectedRouter;

      render(
        <HostsEnrolledCard
          counts={counts}
          totalHostCount={22070}
          builtInLabels={builtInLabels}
          currentTeamId={3}
          router={router}
        />
      );

      const macButton = screen.getByRole("button", { name: "macOS hosts" });

      fireEvent.keyDown(macButton, { key: "Enter" });
      fireEvent.keyDown(macButton, { key: " " });

      expect(push).toHaveBeenCalledTimes(2);
      // Links to the macOS built-in label (id 10) while preserving the fleet.
      expect(push).toHaveBeenCalledWith(expect.stringContaining("/labels/10"));
      expect(push).toHaveBeenCalledWith(expect.stringContaining("fleet_id=3"));
    });
  });
});
