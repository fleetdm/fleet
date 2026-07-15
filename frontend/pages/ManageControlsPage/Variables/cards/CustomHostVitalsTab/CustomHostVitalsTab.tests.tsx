import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  createCustomRenderer,
  createMockRouter,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";

import CustomHostVitalsTab from "./CustomHostVitalsTab";

// The tab filters server-side, so intercept GET /custom_host_vitals and return
// the seeded vitals matching the `query` param — by name or the derived
// $FLEET_HOST_VITAL_<id> token — mirroring the backend's search behavior.
const SEED = [
  { id: 1, name: "Asset tag", created_at: "", updated_at: "" },
  { id: 2, name: "Department", created_at: "", updated_at: "" },
  { id: 3, name: "Purchase date", created_at: "", updated_at: "" },
];

const customHostVitalsHandler = http.get(
  baseUrl("/custom_host_vitals"),
  ({ request }) => {
    const q = (new URL(request.url).searchParams.get("query") ?? "")
      .trim()
      .toLowerCase();
    const filtered = q
      ? SEED.filter(
          (v) =>
            v.name.toLowerCase().includes(q) ||
            `$fleet_host_vital_${v.id}`.includes(q)
        )
      : SEED;
    return HttpResponse.json({
      custom_host_vitals: filtered,
      count: filtered.length,
      meta: { has_next_results: false, has_previous_results: false },
    });
  }
);

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

  beforeEach(() => {
    mockServer.use(customHostVitalsHandler);
  });

  it("pre-fills the search input from the URL and filters the list on mount", async () => {
    const props = makeProps({ query: "depart" });
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(() => {
      const searchInput = screen.getByPlaceholderText(
        "Search by name"
      ) as HTMLInputElement;
      expect(searchInput.value).toBe("depart");
    });

    await waitFor(() => {
      expect(screen.getByText("Department")).toBeInTheDocument();
      expect(screen.queryByText("Asset tag")).not.toBeInTheDocument();
      expect(screen.queryByText("Purchase date")).not.toBeInTheDocument();
    });
  });

  it("matches the variable token as well as the name", async () => {
    // "Department" seeds as id 2 -> token $FLEET_HOST_VITAL_2.
    const props = makeProps({ query: "fleet_host_vital_2" });
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(() => {
      expect(screen.getByText("Department")).toBeInTheDocument();
      expect(screen.queryByText("Asset tag")).not.toBeInTheDocument();
    });
  });

  it("renders the full list when no search param is present", async () => {
    const props = makeProps();
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(() => {
      expect(screen.getByText("Asset tag")).toBeInTheDocument();
      expect(screen.getByText("Department")).toBeInTheDocument();
      expect(screen.getByText("Purchase date")).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText(
      "Search by name"
    ) as HTMLInputElement;
    expect(searchInput.value).toBe("");
  });
});
