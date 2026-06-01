// State is passed in through tableConfig which is tested in the parent component's tests (SelfService.tests.tsx)

import React from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";

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
  per_page: 20,
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
  isEmptySearch: false,
  router: createMockRouter(),
  pathname: "/device/software",
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

  it("calls router.push when category dropdown changes", async () => {
    const mockRouter = createMockRouter();
    const props = createTestProps({ router: mockRouter });
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();

    render(<SelfServiceCard {...props} />);

    const dropdown = screen.getByRole("combobox");
    await user.click(dropdown);

    expect(mockRouter.push).toHaveBeenCalled();
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
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, category_id: 42 },
      enhancedSoftware: [
        {
          ...createMockDeviceSoftware({ name: "installed-app" }),
          ui_status: "installed",
        },
        {
          ...createMockDeviceSoftware({ name: "uninstalled-app" }),
          ui_status: "uninstalled",
        },
        {
          ...createMockDeviceSoftware({ name: "another-uninstalled-app" }),
          ui_status: "uninstalled",
        },
      ],
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    // Without a matching custom category, filter returns all items unchanged,
    // so 2 of 3 are eligible.
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

  it("renders empty search state when isEmptySearch is true", () => {
    const props = createTestProps({
      isEmptySearch: true,
      enhancedSoftware: [],
      queryParams: { ...DEFAULT_QUERY_PARAMS, query: "nonexistent" },
    });
    const render = createCustomRenderer({ withBackendMock: true });

    render(<SelfServiceCard {...props} />);

    expect(screen.getByText("No items match your search")).toBeInTheDocument();
    expect(
      screen.getByText(/Not finding what you're looking for/)
    ).toBeInTheDocument();

    const contactLink = screen.getAllByRole("link", {
      name: /Reach out to IT/i,
    });
    expect(contactLink[0]).toHaveAttribute("href", props.contactUrl);
  });
});
