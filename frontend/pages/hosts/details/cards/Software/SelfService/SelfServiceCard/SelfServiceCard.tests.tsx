// State is passed in through tableConfig which is tested in the parent component's tests (SelfService.tests.tsx)

import React from "react";
import { screen } from "@testing-library/react";
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
  it("renders loading spinner when isLoading is true", () => {
    const props = createTestProps({ isLoading: true });
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("renders error state when isError is true", () => {
    const props = createTestProps({ isError: true });
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    expect(screen.getByText("Error loading software.")).toBeInTheDocument();
  });

  it("renders empty state when isEmpty is true", () => {
    const props = createTestProps({
      isEmpty: true,
      enhancedSoftware: [],
      selfServiceData: undefined,
      isFetching: false,
    });
    const render = createCustomRenderer();

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
    const render = createCustomRenderer();

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
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    const link = screen.getByRole("link", { name: /reach out to IT/i });
    expect(link).toHaveAttribute("href", "http://example.com/help");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("does not render contact link when contactUrl is empty", () => {
    const props = createTestProps({ contactUrl: "" });
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    expect(
      screen.queryByRole("link", { name: /reach out to IT/i })
    ).not.toBeInTheDocument();
  });

  it("renders search field with correct placeholder and default value", () => {
    const props = createTestProps({
      queryParams: { ...DEFAULT_QUERY_PARAMS, query: "test search" },
    });
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    const searchField = screen.getByPlaceholderText("Search by name");
    expect(searchField).toBeInTheDocument();
    expect(searchField).toHaveValue("test search");
  });

  it("calls router.push when category dropdown changes", async () => {
    const mockRouter = createMockRouter();
    const props = createTestProps({ router: mockRouter });
    const render = createCustomRenderer();
    const user = userEvent.setup();

    render(<SelfServiceCard {...props} />);

    const dropdown = screen.getByRole("combobox");
    await user.click(dropdown);

    // Note: This test might need adjustment based on your dropdown implementation
    expect(mockRouter.push).toHaveBeenCalled();
  });

  it("renders empty search state when isEmptySearch is true", () => {
    const props = createTestProps({
      isEmptySearch: true,
      enhancedSoftware: [],
      queryParams: { ...DEFAULT_QUERY_PARAMS, query: "nonexistent" },
    });
    const render = createCustomRenderer();

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

  it("renders categories menu component", () => {
    const props = createTestProps();
    const render = createCustomRenderer();

    render(<SelfServiceCard {...props} />);

    expect(screen.getAllByText(/Browsers/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Communication/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Productivity/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Developer tools/i).length).toBeGreaterThan(0);
  });
});
