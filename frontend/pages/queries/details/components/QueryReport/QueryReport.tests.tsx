import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";

import QueryReport from "./QueryReport";

describe("QueryReport", () => {
  it("Renders cell data normally when not longer than 300 chars", () => {
    const [isClipped, queryReport] = [
      false,
      {
        query_id: 1,
        results: [
          {
            host_id: 1,
            host_name: "host1",
            last_fetched: "2020-01-01",
            columns: { col1: "value1", col2: "value2" },
          },
          {
            host_id: 2,
            host_name: "host2",
            last_fetched: "2020-01-01",
            columns: { col1: "value3", col2: "value4" },
          },
        ],
        report_clipped: false,
      },
    ];
    render(<QueryReport {...{ isClipped, queryReport }} />);

    expect(screen.getByText(/value2/)).toBeInTheDocument();
    expect(screen.queryByText("truncated")).not.toBeInTheDocument();
    expect(screen.queryByText(/\.\.\./)).not.toBeInTheDocument();
  });

  it("Renders truncated cell data when not longer than 300 chars", () => {
    const [isClipped, queryReport] = [
      false,
      {
        query_id: 1,
        results: [
          {
            host_id: 1,
            host_name: "host1",
            last_fetched: "2021-01-01",
            columns: { col1: "value1", col2: "value2" },
          },
          {
            host_id: 2,
            host_name: "host2",
            last_fetched: "2021-01-01",
            columns: {
              col1: "value1",
              col2:
                "/Applications/Docker.app/Contents/MacOS/Docker Desktop.app/Contents/Frameworks/Docker Desktop Helper (GPU).app/Contents/MacOS/Docker Desktop Helper (GPU) --type=gpu-process --user-data-dir=/Users/reed/Library/Application Support/Docker Desktop --gpu-preferences=UAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAABgAAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJgEAAAAAAAAmAQAAAAAAACIAQAAMAAAAIABAAAAAAAAiAEAAAAAAACQAQAAAAAAAJgBAAAAAAAAoAEAAAAAAACoAQAAAAAAALABAAAAAAAAuAEAAAAAAADAAQAAAAAAAMgBAAAAAAAA0AEAAAAAAADYAQAAAAAAAOABAAAAAAAA6AEAAAAAAADwAQAAAAAAAPgBAAAAAAAAAAIAAAAAAAAIAgAAAAAAABACAAAAAAAAGAIAAAAAAAAgAgAAAAAAACgCAAAAAAAAMAIAAAAAAAA4AgAAAAAAAEACAAAAAAAASAIAAAAAAABQAgAAAAAAAFgCAAAAAAAAYAIAAAAAAABoAgAAAAAAAHACAAAAAAAAeAIAAAAAAACAAgAAAAAAAIgCAAAAAAAAkAIAAAAAAACYAgAAAAAAAKACAAAAAAAAqAIAAAAAAACwAgAAAAAAALgCAAAAAAAAwAIAAAAAAADIAgAAAAAAANACAAAAAAAA2AIAAAAAAADgAgAAAAAAAOgCAAAAAAAA8AIAAAAAAAD4AgAAAAAAABAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAHAAAAEAAAAAAAAAAAAAAACAAAABAAAAAAAAAAAAAAAAkAAAAQAAAAAAAAAAAAAAALAAAAEAAAAAAAAAAAAAAADAAAABAAAAAAAAAAAAAAAA4AAAAQAAAAAAAAAAAAAAAPAAAAEAAAAAAAAAABAAAAAAAAABAAAAAAAAAAAQAAAAcAAAAQAAAAAAAAAAEAAAAIAAAAEAAAAAAAAAABAAAACQAAABAAAAAAAAAAAQAAAAsAAAAQAAAAAAAAAAEAAAAMAAAAEAAAAAAAAAABAAAADgAAABAAAAAAAAAAAQAAAA8AAAAQAAAAAAAAAAQAAAAAAAAAEAAAAAAAAAAEAAAABwAAABAAAAAAAAAABAAAAAgAAAAQAAAAAAAAAAQAAAAJAAAAEAAAAAAAAAAEAAAACwAAABAAAAAAAAAABAAAAAwAAAAQAAAAAAAAAAQAAAAOAAAAEAAAAAAAAAAEAAAADwAAABAAAAAAAAAABwAAAAAAAAAQAAAAAAAAAAcAAAAHAAAAEAAAAAAAAAAHAAAACAAAABAAAAAAAAAABwAAAAkAAAAQAAAAAAAAAAcAAAALAAAAEAAAAAAAAAAHAAAADAAAABAAAAAAAAAABwAAAA4AAAAQAAAAAAAAAAcAAAAPAAAAEAAAAAAAAAAIAAAAAAAAABAAAAAAAAAACAAAAAcAAAAQAAAAAAAAAAgAAAAIAAAAEAAAAAAAAAAIAAAACQAAABAAAAAAAAAACAAAAAsAAAAQAAAAAAAAAAgAAAAMAAAAEAAAAAAAAAAIAAAADgAAABAAAAAAAAAACAAAAA8AAAAQAAAAAAAAAAoAAAAAAAAAEAAAAAAAAAAKAAAABwAAABAAAAAAAAAACgAAAAgAAAAQAAAAAAAAAAoAAAAJAAAAEAAAAAAAAAAKAAAACwAAABAAAAAAAAAACgAAAAwAAAAQAAAAAAAAAAoAAAAOAAAAEAAAAAAAAAAKAAAADwAAAAgAAAAAAAAACAAAAAAAAAA= --shared-files --field-trial-handle=1718379636,11537667402821735008,10648286844359859266,131072 --disable-features=PlzServiceWorker,SpareRendererForSitePerProcess --seatbelt-client=49",
            },
          },
        ],
        report_clipped: false,
      },
    ];
    render(<QueryReport {...{ isClipped, queryReport }} />);

    expect(screen.getByText(/value2/)).toBeInTheDocument();
    expect(screen.getByText(/(truncated)/)).toBeInTheDocument();
    expect(screen.getByText(/\.\.\./)).toBeInTheDocument();
  });
  it("Renders a tooltip informing the user that the report is clipped.", async () => {
    const [isClipped, queryReport] = [
      true,
      {
        query_id: 1,
        results: [
          {
            host_id: 1,
            host_name: "host1",
            last_fetched: "2021-01-01",
            columns: { col1: "value1", col2: "value2" },
          },
          {
            host_id: 2,
            host_name: "host2",
            last_fetched: "2021-01-01",
            columns: { col1: "value1", col2: "value2" },
          },
        ],
        report_clipped: true,
      },
    ];
    render(<QueryReport {...{ isClipped, queryReport }} />);

    await fireEvent.mouseEnter(screen.getByText(/\d+ result/));

    expect(
      screen.getByText(
        /Fleet has retained a sample of early results for reference/
      )
    ).toBeInTheDocument();
  });
});
