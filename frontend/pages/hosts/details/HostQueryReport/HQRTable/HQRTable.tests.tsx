import React from "react";

import { render, screen } from "@testing-library/react";

import HQRTable, { IHQRTable } from "./HQRTable";

describe("HQRTable component", () => {
  it("Renders results normally when they are present", () => {
    const testData: IHQRTable[] = [
      {
        queryName: "testQuery0",
        queryDescription: "testDescription0",
        hostName: "testHost0",
        rows: [
          {
            build_distro: "10.14",
            build_platform: "darwin",
            config_hash: "111111111111111111111111",
            config_valid: "1",
            extensions: "active",
            instance_id: "2f7a7b8e-8f35-4fa8-9e8b-1111111111111111",
            pid: "575",
            platform_mask: "21",
            start_time: "1711512878",
            uuid: "gggggg-4568-5BD9-9F1C-6D2E701FAB5C",
            version: "5.11.0",
            watcher: "574",
          },
        ],
        reportClipped: false,
        lastFetched: "2021-09-01T00:00:00Z",
        onShowQuery: jest.fn(),
        isLoading: false,
      },
    ];

    testData.forEach((tableProps) => {
      render(<HQRTable {...tableProps} />);
      expect(screen.getByText("1 result")).toBeInTheDocument();
      expect(screen.getByText("Last fetched")).toBeInTheDocument();
      tableProps.rows.forEach((row) => {
        Object.entries(row).forEach(([col, val]) => {
          expect(screen.getByText(col)).toBeInTheDocument();
          expect(screen.getByText(val)).toBeInTheDocument();
        });
      });
    });
  });

  it("Renders the 'collecting results' empty state when results have never been collected.", () => {
    const testData: IHQRTable[] = [
      {
        queryName: "testQuery0",
        queryDescription: "testDescription0",
        hostName: "testHost0",
        rows: [],
        reportClipped: false,
        lastFetched: null,
        onShowQuery: jest.fn(),
        isLoading: false,
      },
    ];

    testData.forEach((tableProps) => {
      render(<HQRTable {...tableProps} />);
      expect(screen.queryByText("Last fetched")).toBeNull();
      expect(screen.getByText("Collecting results...")).toBeInTheDocument();
    });
  });

  it("Renders the 'report clipped' empty state when reporting for this query has been paused and there are no existing results.", () => {
    const testData: IHQRTable[] = [
      {
        queryName: "testQuery0",
        queryDescription: "testDescription0",
        hostName: "testHost0",
        rows: [],
        reportClipped: true,
        lastFetched: "2021-09-01T00:00:00Z",
        onShowQuery: jest.fn(),
        isLoading: false,
      },
    ];

    testData.forEach((tableProps) => {
      render(<HQRTable {...tableProps} />);
      expect(screen.queryByText("Last fetched")).toBeNull();
      expect(screen.getByText("Report clipped")).toBeInTheDocument();
    });
  });

  it("Renders the 'nothing to report' empty state when the query has run and there are no results.", () => {
    const testData: IHQRTable[] = [
      {
        queryName: "testQuery0",
        queryDescription: "testDescription0",
        hostName: "testHost0",
        rows: [],
        reportClipped: false,
        lastFetched: "2021-09-01T00:00:00Z",
        onShowQuery: jest.fn(),
        isLoading: false,
      },
    ];

    testData.forEach((tableProps) => {
      render(<HQRTable {...tableProps} />);
      expect(screen.queryByText("Last fetched")).toBeNull();
      expect(screen.getByText("Nothing to report")).toBeInTheDocument();
    });
  });
});
