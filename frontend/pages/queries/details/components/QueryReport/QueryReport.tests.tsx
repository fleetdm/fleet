import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import FileSaver from "file-saver";
import React from "react";

import { notify } from "components/ToastNotification";
import { IUserSettings } from "interfaces/config";
import { IUser } from "interfaces/user";
import usersAPI from "services/entities/users";
import { renderWithAppContext, renderWithSetup } from "test/test-utils";

import QueryReport from "./QueryReport";

const tableProps = {
  pageIndex: 0,
  pageSize: 50,
  searchQuery: "",
  sortHeader: "host_name",
  sortDirection: "asc",
  onQueryChange: jest.fn(),
  loadAllResults: jest.fn().mockResolvedValue([]),
};

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
        count: 2,
      },
    ];
    render(
      <QueryReport
        queryId={1}
        {...tableProps}
        {...{ isClipped, queryReport }}
      />
    );

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
        count: 2,
      },
    ];
    render(
      <QueryReport
        queryId={1}
        {...tableProps}
        {...{ isClipped, queryReport }}
      />
    );

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
        count: 2,
      },
    ];
    const { user } = renderWithSetup(
      <QueryReport
        queryId={1}
        {...tableProps}
        {...{ isClipped, queryReport }}
      />
    );

    await user.hover(screen.getByText(/\d+ result/));

    await waitFor(() => {
      expect(
        screen.getByText(
          /This report is full. Hosts already in the report keep updating/
        )
      ).toBeInTheDocument();
    });
  });

  it("Shows a search-specific empty state when a search matches nothing", () => {
    render(
      <QueryReport
        queryId={1}
        {...tableProps}
        searchQuery="nomatch"
        queryReport={{
          query_id: 1,
          results: [],
          report_clipped: false,
          count: 0,
        }}
      />
    );

    expect(screen.getByText("No results match")).toBeInTheDocument();
    expect(screen.queryByText("Nothing to report yet")).not.toBeInTheDocument();
  });

  it("Maps the Host table column to the host_name API order key", async () => {
    const onQueryChange = jest.fn();
    const queryReport = {
      query_id: 1,
      results: [
        {
          host_id: 1,
          host_name: "host1",
          last_fetched: "2021-01-01",
          columns: { col1: "value1" },
        },
      ],
      report_clipped: false,
      count: 1,
    };
    const { user } = renderWithSetup(
      <QueryReport
        queryId={1}
        {...tableProps}
        onQueryChange={onQueryChange}
        sortHeader="host_name"
        sortDirection="asc"
        queryReport={queryReport}
      />
    );

    // The initial query change carries the mapped-back API key.
    await waitFor(() => expect(onQueryChange).toHaveBeenCalled());
    expect(onQueryChange.mock.calls[0][0]).toMatchObject({
      sortHeader: "host_name",
      sortDirection: "asc",
    });

    await user.click(screen.getByText("Host"));
    await waitFor(() => {
      const { calls } = onQueryChange.mock;
      expect(calls[calls.length - 1][0]).toMatchObject({
        sortHeader: "host_name",
        sortDirection: "desc",
      });
    });
  });

  it("Exports every result with columns from the full export, not the page", async () => {
    const saveAs = jest
      .spyOn(FileSaver, "saveAs")
      .mockImplementation(() => undefined);
    const loadAllResults = jest.fn().mockResolvedValue([
      {
        host_id: 1,
        host_name: "host1",
        last_fetched: "2021-01-01",
        columns: { col1: "value1" },
      },
      {
        host_id: 2,
        host_name: "host2",
        last_fetched: "2021-01-01",
        columns: { col1: "value2", only_on_page_two: "x" },
      },
    ]);
    const queryReport = {
      query_id: 1,
      results: [
        {
          host_id: 1,
          host_name: "host1",
          last_fetched: "2021-01-01",
          columns: { col1: "value1" },
        },
      ],
      report_clipped: false,
      count: 2,
    };
    const { user } = renderWithSetup(
      <QueryReport
        queryId={1}
        {...tableProps}
        loadAllResults={loadAllResults}
        queryReport={queryReport}
      />
    );

    await user.click(screen.getByRole("button", { name: /Export results/ }));

    await waitFor(() => expect(saveAs).toHaveBeenCalledTimes(1));
    expect(loadAllResults).toHaveBeenCalledTimes(1);
    const file = saveAs.mock.calls[0][0] as File;
    // jsdom's File has no text(); read it the old way.
    const csv = await new Promise<string>((resolve) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.readAsText(file);
    });
    expect(csv).toContain("only_on_page_two");
    expect(csv).toContain("value2");
    saveAs.mockRestore();
  });
});

describe("Report column preferences", () => {
  const queryReport = {
    query_id: 6,
    results: [
      {
        host_id: 1,
        host_name: "host1",
        last_fetched: "2020-01-01",
        columns: { name: "example", version: "1" },
      },
    ],
    report_clipped: false,
    count: 1,
  };

  const setup = (
    settings: IUserSettings = {
      hidden_host_columns: ["osquery_version"],
      report_column_order: { "7": ["name"] },
    }
  ) => {
    const setUserSettings = jest.fn();
    const view = renderWithAppContext(
      <QueryReport queryId={6} queryReport={queryReport} {...tableProps} />,
      {
        contextValue: {
          currentUser: { id: 1 } as IUser,
          userSettings: settings,
          setUserSettings,
        },
      }
    );
    return { ...view, setUserSettings };
  };

  it("saves a report-specific order while preserving other preferences", async () => {
    const update = jest
      .spyOn(usersAPI, "update")
      .mockResolvedValue({} as IUser);
    const { setUserSettings } = setup();
    fireEvent.click(screen.getByRole("button", { name: "Column preferences" }));
    expect(
      screen.getByRole("button", { name: "Move up: Host" })
    ).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Move up: name" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(setUserSettings).toHaveBeenCalled());
    expect(update).toHaveBeenCalledWith(1, {
      settings: {
        report_column_order: {
          "6": ["Host", "name", "last_fetched", "version"],
        },
      },
    });
    const latest = {
      hidden_host_columns: ["hostname"],
      report_column_order: { "8": ["version", "Host"] },
    };
    expect(setUserSettings.mock.calls[0][0](latest)).toEqual({
      ...latest,
      report_column_order: {
        ...latest.report_column_order,
        "6": ["Host", "name", "last_fetched", "version"],
      },
    });
    update.mockRestore();
  });

  it("disables column preferences until user settings load", () => {
    renderWithAppContext(
      <QueryReport queryId={6} queryReport={queryReport} {...tableProps} />,
      {
        contextValue: {
          currentUser: { id: 1 } as IUser,
          userSettings: undefined,
        },
      }
    );
    expect(
      screen.getByRole("button", { name: "Column preferences" })
    ).toBeDisabled();
  });

  it("loads saved order, ignores removed columns, and appends new columns", () => {
    setup({
      hidden_host_columns: [],
      report_column_order: { "6": ["name", "removed", "Host", "name"] },
    });
    const headers = screen.getAllByRole("columnheader");
    expect(headers[0]).toHaveTextContent("name");
    expect(headers[1]).toHaveTextContent("Host");
    expect(headers[2]).toHaveTextContent("Last fetched");
    expect(headers[3]).toHaveTextContent("version");
  });

  it("keeps the dialog and original preferences when saving fails", async () => {
    const update = jest
      .spyOn(usersAPI, "update")
      .mockRejectedValue(new Error("offline"));
    const error = jest.spyOn(notify, "error").mockImplementation(jest.fn());
    const { setUserSettings } = setup();
    fireEvent.click(screen.getByRole("button", { name: "Column preferences" }));
    fireEvent.click(screen.getByRole("button", { name: "Move up: name" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(error).toHaveBeenCalled());
    expect(setUserSettings).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    expect(screen.getAllByRole("columnheader")[1]).toHaveTextContent(
      "Last fetched"
    );
    update.mockRestore();
    error.mockRestore();
  });

  it("retains absent column positions and keyboard focus", async () => {
    const update = jest
      .spyOn(usersAPI, "update")
      .mockResolvedValue({} as IUser);
    setup({
      hidden_host_columns: [],
      report_column_order: {
        "6": ["Host", "absent", "name", "last_fetched", "version"],
      },
    });
    fireEvent.click(screen.getByRole("button", { name: "Column preferences" }));
    fireEvent.click(screen.getByRole("button", { name: "Move up: name" }));
    fireEvent.click(screen.getByRole("button", { name: "Move up: name" }));
    expect(
      screen.getByRole("button", { name: "Move down: name" })
    ).toHaveFocus();
    expect(
      screen.getByRole("status", { name: "Column position" })
    ).toHaveTextContent("name moved to position 1 of 4");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(update).toHaveBeenCalled());
    expect(update.mock.calls[0][1].settings?.report_column_order?.[6]).toEqual([
      "name",
      "absent",
      "Host",
      "last_fetched",
      "version",
    ]);
    update.mockRestore();
  });

  it("exports Host at its saved position", async () => {
    const saveAs = jest
      .spyOn(FileSaver, "saveAs")
      .mockImplementation(jest.fn());
    const load = tableProps.loadAllResults.mockResolvedValue(
      queryReport.results
    );
    setup({
      hidden_host_columns: [],
      report_column_order: { "6": ["name", "Host", "version", "last_fetched"] },
    });
    fireEvent.click(screen.getByRole("button", { name: /Export results/ }));
    await waitFor(() => expect(saveAs).toHaveBeenCalled());
    const csv = await new Promise<string>((resolve) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.readAsText(saveAs.mock.calls[0][0] as File);
    });
    expect(csv.split("\n")[0]).toBe(
      '"name","host_display_name","version","last_fetched"'
    );
    expect(csv.split("\n")[1]).toContain('"example","host1","1"');
    saveAs.mockRestore();
    load.mockResolvedValue([]);
  });

  it("keeps keyboard focus in the dialog and restores it on close", () => {
    setup();
    const trigger = screen.getByRole("button", { name: "Column preferences" });
    trigger.focus();
    fireEvent.click(trigger);
    expect(
      screen.getByRole("button", { name: "Move down: Host" })
    ).toHaveFocus();

    const cancel = screen.getByRole("button", { name: "Cancel" });
    cancel.focus();
    fireEvent.keyDown(cancel, { key: "Tab" });
    const dialog = cancel.closest(".modal__modal_container") as HTMLElement;
    expect(dialog).toContainElement(document.activeElement as HTMLElement);
    expect(document.activeElement).not.toBe(cancel);
    fireEvent.keyDown(document.activeElement as HTMLElement, {
      key: "Tab",
      shiftKey: true,
    });
    expect(cancel).toHaveFocus();

    fireEvent.click(cancel);
    expect(trigger).toHaveFocus();
  });

  it("cancels without saving", () => {
    const update = jest.spyOn(usersAPI, "update");
    setup();
    fireEvent.click(screen.getByRole("button", { name: "Column preferences" }));
    fireEvent.click(screen.getByRole("button", { name: "Move up: name" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(update).not.toHaveBeenCalled();
    expect(screen.getAllByRole("columnheader")[1]).toHaveTextContent(
      "Last fetched"
    );
    update.mockRestore();
  });
});
