// State is passed in through tableConfig which is tested in the parent component's tests (SelfService.tests.tsx)

import React from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { http, HttpResponse } from "msw";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { baseUrl } from "test/default-handlers";
import { listDeviceSelfServiceCategoriesHandler } from "test/handlers/self-service-categories-handlers";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import { createMockHostSoftwarePackage } from "__mocks__/hostMock";
import { SoftwareCategory } from "interfaces/software";

import SelfServiceCard, {
  SelfServiceQueryParams,
  ISelfServiceCardProps,
} from "./SelfServiceCard";

const createMockTableConfig = () => [
  {
    title: "Name",
    accessor: "name",
    disableHidden: false,
  },
  {
    title: "Status",
    accessor: "status",
    disableHidden: false,
  },
  {
    title: "Actions",
    accessor: "actions",
    disableHidden: false,
  },
];

const DEFAULT_QUERY_PARAMS: SelfServiceQueryParams = {
  page: 0,
  query: "",
  order_key: "name",
  order_direction: "asc",
  per_page: 9999,
  category_id: undefined,
};

const createTestProps = (
  overrides: Partial<ISelfServiceCardProps> = {}
): ISelfServiceCardProps => ({
  contactUrl: "http://example.com/contact",
  deviceToken: "test-device-token",
  queryParams: DEFAULT_QUERY_PARAMS,
  enhancedSoftware: [
    { ...createMockDeviceSoftware({ name: "test1" }), ui_status: "installed" },
    { ...createMockDeviceSoftware({ name: "test2" }), ui_status: "installed" },
    { ...createMockDeviceSoftware({ name: "test3" }), ui_status: "installed" },
  ],
  selfServiceData: {
    count: 3,
    software: [
      createMockDeviceSoftware({ name: "test1" }),
      createMockDeviceSoftware({ name: "test2" }),
      createMockDeviceSoftware({ name: "test3" }),
    ],
    meta: {
      has_previous_results: false,
      has_next_results: false,
    },
  },
  tableConfig: createMockTableConfig(),
  isLoading: false,
  isError: false,
  isFetching: false,
  isEmpty: false,
  router: createMockRouter(),
  pathname: "/device/software",
  onClickInstallAction: jest.fn(),
  ...overrides,
});

describe("SelfServiceCard", () => {
  it("renders loading spinner when isLoading is true", async () => {
    const props = createTestProps({ isLoading: true });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    // Spinner has a built-in anti-flash delay, so wait for it to appear.
    expect(await screen.findByTestId("spinner")).toBeInTheDocument();
  });

  it("renders error state when isError is true", () => {
    const props = createTestProps({ isError: true });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(screen.getByText("Error loading software")).toBeInTheDocument();
  });

  it("renders empty state when isEmpty is true", () => {
    const props = createTestProps({
      isEmpty: true,
      enhancedSoftware: [],
      selfServiceData: undefined,
      isFetching: false,
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(
      screen.getByText("No self-service software available yet")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Your organization didn’t add any self-service software./i
      )
    ).toBeInTheDocument();
  });

  it("renders self-service card with header and subheader", () => {
    const props = createTestProps();
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(screen.getByText("Self-service")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Install organization-approved apps provided by your IT department/
      )
    ).toBeInTheDocument();
  });

  it("renders contact link when contactUrl is provided", () => {
    const props = createTestProps({ contactUrl: "http://example.com/help" });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    const link = screen.getByRole("link", { name: /reach out to IT/i });
    expect(link).toHaveAttribute("href", "http://example.com/help");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("does not render contact link when contactUrl is empty", () => {
    const props = createTestProps({ contactUrl: "" });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(
      screen.queryByRole("link", { name: /reach out to IT/i })
    ).not.toBeInTheDocument();
  });

  it("hides the category dropdown when the categories list is empty", async () => {
    // Default handler is `emptyDeviceSelfServiceCategoriesHandler` — no .use() needed.
    const props = createTestProps();
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    // Give the categories query a tick to resolve; the trigger should never appear.
    await waitFor(() => {
      expect(
        screen.queryByRole("button", { expanded: false })
      ).not.toBeInTheDocument();
    });
  });

  it("auto-clears a stale category_id from the URL when categories load empty", async () => {
    const pushSpy = jest.fn();
    const mockRouter = createMockRouter({ push: pushSpy });
    const props = createTestProps({
      router: mockRouter,
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 99 },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    // At least one push must drop `category_id` (the auto-clear); other pushes
    // from table initialization etc. may also fire, so we don't pin to the
    // first call.
    await waitFor(() => {
      expect(pushSpy).toHaveBeenCalledWith(
        expect.not.stringContaining("category_id")
      );
    });
  });

  it("auto-clears a category_id that isn't in the loaded categories list", async () => {
    // Bookmarked link to a since-deleted category — list is non-empty but id
    // 99 isn't in it. Without recovery the trigger would label "All" while the
    // table sat empty.
    mockServer.use(
      listDeviceSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
    );
    const pushSpy = jest.fn();
    const mockRouter = createMockRouter({ push: pushSpy });
    const props = createTestProps({
      router: mockRouter,
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 99 },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    await waitFor(() => {
      expect(pushSpy).toHaveBeenCalledWith(
        expect.not.stringContaining("category_id")
      );
    });
  });

  it("renders search field with correct placeholder and default value", () => {
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, query: "test search" },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    const searchField = screen.getByPlaceholderText("Search by name");
    expect(searchField).toBeInTheDocument();
    expect(searchField).toHaveValue("test search");
  });

  it("calls router.push with the selected category_id when a category is picked", async () => {
    mockServer.use(
      listDeviceSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
    );
    // Override `push` with a fresh jest.fn() — DEFAULT_MOCK_ROUTER's spies are
    // shared across tests, so a stale call from another test would otherwise
    // satisfy toHaveBeenCalled().
    const pushSpy = jest.fn();
    const mockRouter = createMockRouter({ push: pushSpy });
    const props = createTestProps({ router: mockRouter });
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();

    render(<SelfServiceCard {...props} />);

    // Wait for the categories query to resolve so the CategoryFilter mounts
    // (it's gated on categories.length > 0 in SelfServiceFilters).
    await user.click(await screen.findByRole("button", { expanded: false }));
    const option = await screen.findByText("🌎 Browsers");
    await user.click(option);

    expect(pushSpy).toHaveBeenCalledWith(
      expect.stringContaining("category_id=1")
    );
  });

  it("renders the install-all button enabled when 'All' is selected and items are eligible", () => {
    const props = createTestProps({
      enhancedSoftware: [
        {
          ...createMockDeviceSoftware({ name: "uninstalled-app" }),
          ui_status: "uninstalled",
        },
      ],
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    const button = screen.getByRole("button", { name: /Install all/i });
    expect(button).toBeInTheDocument();
    expect(button).toBeEnabled();
  });

  it("renders the install-all button with the uninstalled count when a category is selected", async () => {
    mockServer.use(
      listDeviceSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
    );
    const browserPackage = createMockHostSoftwarePackage({
      categories: (["🌎 Browsers"] as string[]) as SoftwareCategory[],
    });
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 1 },
      enhancedSoftware: [
        {
          ...createMockDeviceSoftware({
            name: "installed-app",
            software_package: browserPackage,
          }),
          ui_status: "installed",
        },
        {
          ...createMockDeviceSoftware({
            name: "uninstalled-app",
            software_package: browserPackage,
          }),
          ui_status: "uninstalled",
        },
        {
          ...createMockDeviceSoftware({
            name: "another-uninstalled-app",
            software_package: browserPackage,
          }),
          ui_status: "uninstalled",
        },
      ],
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    // 2 of 3 items in Browsers are uninstalled.
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /Install all \(2\)/i })
      ).toBeInTheDocument();
    });
  });

  it("disables the install-all button when an item in the category is in-progress", async () => {
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 42 },
      enhancedSoftware: [
        {
          ...createMockDeviceSoftware({ name: "uninstalled-app" }),
          ui_status: "uninstalled",
        },
        {
          ...createMockDeviceSoftware({ name: "in-progress-app" }),
          ui_status: "installing",
        },
      ],
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    const button = await screen.findByRole("button", {
      name: /Install all/i,
    });
    expect(button).toBeDisabled();
  });

  it("posts to install_all and fires onInstallAllSuccess when the confirm modal is submitted", async () => {
    let installAllCalled = false;
    let installAllUrl = "";
    mockServer.use(
      http.post(
        baseUrl("/device/:token/software/install_all"),
        ({ request }) => {
          installAllCalled = true;
          installAllUrl = request.url;
          return new HttpResponse(null, { status: 202 });
        }
      )
    );
    const onInstallAllSuccess = jest.fn();
    const props = createTestProps({
      onInstallAllSuccess,
      enhancedSoftware: [
        {
          ...createMockDeviceSoftware({ name: "uninstalled-app" }),
          ui_status: "uninstalled",
        },
      ],
    });
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();

    render(<SelfServiceCard {...props} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(1\)/i })
    );
    // The confirm button inside the modal is labeled "Install all" (no count).
    await user.click(
      await screen.findByRole("button", { name: /^Install all$/i })
    );

    await waitFor(() => {
      expect(installAllCalled).toBe(true);
      expect(onInstallAllSuccess).toHaveBeenCalled();
    });
    // "All" selected → no category_id should be on the query string.
    expect(installAllUrl).not.toContain("category_id");
  });

  it("does not render the install-all button on the mobile view", () => {
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 42 },
      isMobileView: true,
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(
      screen.queryByRole("button", { name: /Install all/i })
    ).not.toBeInTheDocument();
  });

  it("renders empty search state when the search query yields no rows", () => {
    const props = createTestProps({
      enhancedSoftware: [],
      queryParams: { ...DEFAULT_QUERY_PARAMS, query: "nonexistent" },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(screen.getByText("No items match your search")).toBeInTheDocument();
    expect(
      screen.getByText(/Not finding what you're looking for/)
    ).toBeInTheDocument();
    // Should NOT render the category-empty copy — search is the active filter.
    expect(
      screen.queryByText("No items in this category")
    ).not.toBeInTheDocument();

    const contactLink = screen.getAllByRole("link", {
      name: /Reach out to IT/i,
    });
    expect(contactLink[0]).toHaveAttribute("href", props.contactUrl);
  });

  it("renders empty-category state when the category filter yields no rows", async () => {
    mockServer.use(
      listDeviceSelfServiceCategoriesHandler([{ id: 1, name: "🌎 Browsers" }])
    );
    // Default enhancedSoftware items have no `categories` entries, so
    // filterSoftwareByCustomCategory returns [] for any category_id.
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 1 },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(
      await screen.findByText("No items in this category")
    ).toBeInTheDocument();
    // Confirm we're NOT falling through to the misleading search-themed copy
    // — no search query was entered.
    expect(
      screen.queryByText("No items match your search")
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("No items match the current search criteria")
    ).not.toBeInTheDocument();
  });
});
