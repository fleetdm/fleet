import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockSoftwareTitle } from "__mocks__/softwareMock";
import { createCustomRenderer } from "test/test-utils";

import InstallSoftwareTable from "./InstallSoftwareTable";

const TITLES = [
  createMockSoftwareTitle({
    id: 1,
    name: "coda",
    display_name: "Superhuman Docs",
  }),
  createMockSoftwareTitle({
    id: 2,
    name: "GUI.delta.guard",
    display_name: "Delta Guard",
  }),
  createMockSoftwareTitle({ id: 3, name: "mmm-no-display-name" }),
];

const EXPECTED_ORDER = [
  "Delta Guard",
  "mmm-no-display-name",
  "Superhuman Docs",
];

const renderTable = () => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(
    <InstallSoftwareTable
      softwareTitles={TITLES}
      onChangeSoftwareSelect={noop}
      platform="macos"
    />
  );
};

const renderedNames = () =>
  screen
    .getAllByRole("row")
    .slice(1)
    .map((row) => EXPECTED_ORDER.find((n) => row.textContent?.includes(n)));

describe("InstallSoftwareTable", () => {
  it("sorts rows by display name, falling back to the title name", () => {
    renderTable();

    expect(renderedNames()).toEqual(EXPECTED_ORDER);
  });

  it("searches by display name", async () => {
    const { user } = renderTable();
    const search = screen.getByPlaceholderText(/search/i);

    await user.type(search, "Superhuman");
    await waitFor(() => expect(renderedNames()).toEqual(["Superhuman Docs"]));

    await user.clear(search);
    await waitFor(() => expect(renderedNames()).toEqual(EXPECTED_ORDER));
  });

  it("searches by the underlying title name", async () => {
    const { user } = renderTable();

    await user.type(screen.getByPlaceholderText(/search/i), "coda");

    await waitFor(() => expect(renderedNames()).toEqual(["Superhuman Docs"]));
  });
});
