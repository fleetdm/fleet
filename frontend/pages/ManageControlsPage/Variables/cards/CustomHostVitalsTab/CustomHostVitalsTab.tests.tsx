import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  createCustomRenderer,
  createMockRouter,
  baseUrl,
} from "test/test-utils";
import mockServer from "test/mock-server";

import CustomHostVitalsTab, {
  CUSTOM_HOST_VITALS_PAGE_SIZE,
} from "./CustomHostVitalsTab";

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

describe("CustomHostVitalsTab - server-side pagination and sort", () => {
  // One full page plus a few, so there's a real second page. Names are
  // zero-padded so page 0 starts at "Vital 01".
  const PAGED_SEED = Array.from(
    { length: CUSTOM_HOST_VITALS_PAGE_SIZE + 3 },
    (_, i) => ({
      id: i + 1,
      name: `Vital ${String(i + 1).padStart(2, "0")}`,
      created_at: "",
      updated_at: "",
    })
  );

  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isGlobalAdmin: true } },
  });

  // Capture the last request's params so we can assert the tab forwards the
  // URL-derived page/sort to the API.
  let lastParams: URLSearchParams | undefined;

  beforeEach(() => {
    lastParams = undefined;
    mockServer.use(
      http.get(baseUrl("/custom_host_vitals"), ({ request }) => {
        const url = new URL(request.url);
        lastParams = url.searchParams;
        const page = parseInt(url.searchParams.get("page") ?? "0", 10);
        const perPage = parseInt(
          url.searchParams.get("per_page") ??
            String(CUSTOM_HOST_VITALS_PAGE_SIZE),
          10
        );
        const start = page * perPage;
        const rows = PAGED_SEED.slice(start, start + perPage);
        return HttpResponse.json({
          custom_host_vitals: rows,
          count: PAGED_SEED.length,
          meta: {
            has_next_results: start + perPage < PAGED_SEED.length,
            has_previous_results: page > 0,
          },
        });
      })
    );
  });

  it("forwards the page and sort params from the URL to the API", async () => {
    const props = makeProps({ page: "1", order_direction: "desc" });
    render(<CustomHostVitalsTab {...props} />);

    await waitFor(() => {
      expect(lastParams?.get("page")).toBe("1");
    });
    expect(lastParams?.get("per_page")).toBe(
      String(CUSTOM_HOST_VITALS_PAGE_SIZE)
    );
    expect(lastParams?.get("order_key")).toBe("name");
    expect(lastParams?.get("order_direction")).toBe("desc");
  });

  it("appends page to the URL when navigating to the next page", async () => {
    const props = makeProps();
    const { user } = render(<CustomHostVitalsTab {...props} />);

    await screen.findByText("Vital 01");
    const nextButton = screen.getByRole("button", { name: /next/i });
    expect(nextButton).toBeEnabled();

    await user.click(nextButton);

    expect(props.router.replace).toHaveBeenCalledWith(
      expect.stringContaining("page=1")
    );
  });
});
