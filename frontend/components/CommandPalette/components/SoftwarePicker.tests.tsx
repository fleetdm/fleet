import React from "react";
import { waitFor } from "@testing-library/react";
import { Command } from "cmdk";
import { createCustomRenderer } from "test/test-utils";

import softwareAPI from "services/entities/software";
import { createMockSoftwareTitle } from "__mocks__/softwareMock";

import SoftwarePicker from "./SoftwarePicker";

// cmdk uses scrollIntoView which JSDOM doesn't implement.
Element.prototype.scrollIntoView = jest.fn();

jest.mock("services/entities/software", () => ({
  __esModule: true,
  default: { getSoftwareTitles: jest.fn() },
}));

const mockedSoftware = softwareAPI as jest.Mocked<typeof softwareAPI>;

const renderPicker = createCustomRenderer({ withBackendMock: true });
// Picker renders Command.Item, which needs a Command root in context.
const renderPickerInCommand = (ui: React.ReactElement) =>
  renderPicker(<Command>{ui}</Command>);

beforeEach(() => {
  mockedSoftware.getSoftwareTitles.mockReset();
  mockedSoftware.getSoftwareTitles.mockResolvedValue({
    count: 0,
    counts_updated_at: null,
    software_titles: [],
    meta: { has_next_results: false, has_previous_results: false },
  });
});

describe("SoftwarePicker", () => {
  it("calls getSoftwareTitles with availableForInstall=true in library scope", async () => {
    renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        scope="library"
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedSoftware.getSoftwareTitles).toHaveBeenCalledWith(
        expect.objectContaining({
          teamId: 5,
          availableForInstall: true,
        })
      );
    });
  });

  it("calls getSoftwareTitles without availableForInstall in inventory scope", async () => {
    renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedSoftware.getSoftwareTitles).toHaveBeenCalledWith(
        expect.objectContaining({
          teamId: 5,
          availableForInstall: undefined,
        })
      );
    });
  });

  it("renders the library empty state when no titles match in the fleet", async () => {
    const { findByText } = renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        scope="library"
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No software in Engineering's library\./)
    ).toBeInTheDocument();
  });

  it("renders the inventory empty state when no titles match in the fleet", async () => {
    const { findByText } = renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No software found in Engineering\./)
    ).toBeInTheDocument();
  });

  it("inventory empty state uses 'in this fleet' for Unassigned", async () => {
    const { findByText } = renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 0, name: "Unassigned" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No software found in this fleet\./)
    ).toBeInTheDocument();
  });

  it("inventory empty state drops the suffix on All fleets", async () => {
    const { findByText } = renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    expect(await findByText(/^No software found\.$/)).toBeInTheDocument();
  });

  it("renders title labels via getDisplayedSoftwareName (display_name and normalization)", async () => {
    mockedSoftware.getSoftwareTitles.mockResolvedValue({
      count: 2,
      counts_updated_at: null,
      software_titles: [
        createMockSoftwareTitle({
          id: 1,
          name: "Zoom.pkg",
          display_name: "Zoom Workplace",
        }),
        createMockSoftwareTitle({
          id: 2,
          name: "Microsoft.CompanyPortal",
        }),
      ],
      meta: { has_next_results: false, has_previous_results: false },
    });

    // Unique search value sidesteps React Query cache pollution from
    // earlier empty-state tests under queryKey [..., "<search>"].
    const { findByText, queryByText } = renderPickerInCommand(
      <SoftwarePicker
        search="display-name-test"
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    expect(await findByText("Zoom Workplace")).toBeInTheDocument();
    expect(await findByText("Company Portal")).toBeInTheDocument();
    expect(queryByText("Zoom.pkg")).not.toBeInTheDocument();
    expect(queryByText("Microsoft.CompanyPortal")).not.toBeInTheDocument();
  });

  it("library empty state uses 'this fleet's library' for Unassigned", async () => {
    const { findByText } = renderPicker(
      <SoftwarePicker
        search=""
        currentTeam={{ id: 0, name: "Unassigned" }}
        scope="library"
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No software in this fleet's library\./)
    ).toBeInTheDocument();
  });
});
