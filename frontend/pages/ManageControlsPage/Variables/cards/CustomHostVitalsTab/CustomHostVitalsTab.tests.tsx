import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";

import CustomHostVitalsTab from "./CustomHostVitalsTab";

// The tab reads its data from the in-memory mock service
// (custom_host_vitals_mock), so these tests don't need the backend mock.
// The mock is seeded with "Asset tag", "Department", and "Purchase date".

const makeProps = (query: Record<string, string> = {}) => ({
  router: createMockRouter(),
  location: {
    pathname: "/controls/variables/custom-host-vitals",
    query,
  },
});

describe("CustomHostVitalsTab - URL-persistent search", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
      },
    },
  });

  it("pre-fills the search input from the URL and filters the list on mount", async () => {
    const props = makeProps({ query: "depart" });
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(
      () => {
        const searchInput = screen.getByPlaceholderText(
          "Search by name"
        ) as HTMLInputElement;
        expect(searchInput.value).toBe("depart");
      },
      { timeout: 3000 }
    );

    await waitFor(
      () => {
        expect(screen.getByText("Department")).toBeInTheDocument();
        expect(screen.queryByText("Asset tag")).not.toBeInTheDocument();
        expect(screen.queryByText("Purchase date")).not.toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it("matches the variable token as well as the name", async () => {
    // "Department" seeds as id 2 -> token $FLEET_HOST_VITAL_2.
    const props = makeProps({ query: "fleet_host_vital_2" });
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(
      () => {
        expect(screen.getByText("Department")).toBeInTheDocument();
        expect(screen.queryByText("Asset tag")).not.toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it("renders the full list when no search param is present", async () => {
    const props = makeProps();
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(
      () => {
        expect(screen.getByText("Asset tag")).toBeInTheDocument();
        expect(screen.getByText("Department")).toBeInTheDocument();
        expect(screen.getByText("Purchase date")).toBeInTheDocument();
      },
      { timeout: 3000 }
    );

    const searchInput = screen.getByPlaceholderText(
      "Search by name"
    ) as HTMLInputElement;
    expect(searchInput.value).toBe("");
  });
});
