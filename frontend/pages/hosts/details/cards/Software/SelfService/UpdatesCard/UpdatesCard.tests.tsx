import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, renderWithSetup } from "test/test-utils";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import { noop } from "lodash";
import {
  IDeviceSoftwareWithUiStatus,
  IHostSoftwareUiStatus,
} from "interfaces/software";
import UpdatesCard from "./UpdatesCard";

const contactUrl = "http://example.com/support";

describe("UpdatesCard", () => {
  const render = createCustomRenderer();

  const createEnhancedSoftware = (
    count = 3,
    overrides = {} as Partial<IDeviceSoftwareWithUiStatus>
  ): IDeviceSoftwareWithUiStatus[] => {
    const uiStatusDefaultOrOverride =
      (overrides?.ui_status as IHostSoftwareUiStatus) || "update_available";
    return Array.from({ length: count }, (_, i) => ({
      ...createMockDeviceSoftware({
        id: 100 + i,
        name: `UpdateApp${i}`,
        ...overrides,
      }),
      ui_status: uiStatusDefaultOrOverride,
    }));
  };

  it("renders the card with header, subheader and 'Update all' button", () => {
    const updates = createEnhancedSoftware(2);

    render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError={false}
      />
    );
    expect(screen.getByText("Updates")).toBeInTheDocument();
    expect(
      screen.getByText(/The following app require updating/i)
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Update all" })).toBeEnabled();
    expect(screen.getByText("reach out to IT")).toHaveAttribute(
      "href",
      contactUrl
    );
  });

  it("renders paginated UpdateSoftwareItem components", () => {
    const updates = createEnhancedSoftware(5); // Should paginate on >3 depending on width.
    render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError={false}
      />
    );
    // Based on the default getUpdatesPageSize, expect 3 on first page (jsdom default width)

    expect(screen.getAllByText("UpdateApp0").length).toBeGreaterThan(0);
    expect(screen.getAllByText("UpdateApp1").length).toBeGreaterThan(0);
    expect(screen.getAllByText("UpdateApp2").length).toBeGreaterThan(0);
    // Should NOT render apps from next page yet
    expect(screen.queryByText("UpdateApp3")).not.toBeInTheDocument();
  });

  it("shows next page of updates with pagination", async () => {
    const updates = createEnhancedSoftware(5);
    const { user } = renderWithSetup(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError={false}
      />
    );
    // Go to next page
    const nextPageButton = screen.getByRole("button", { name: /next/i });
    await user.click(nextPageButton);

    expect(screen.getAllByText("UpdateApp3").length).toBeGreaterThan(0);
    expect(screen.getAllByText("UpdateApp4").length).toBeGreaterThan(0);
    // Previous page update should not be present
    expect(screen.queryByText("UpdateApp0")).not.toBeInTheDocument();
  });

  it("disables the 'Update all' button if all updating", () => {
    const updates = createEnhancedSoftware(2, { ui_status: "updating" });
    render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError={false}
      />
    );
    const button = screen.getByRole("button", { name: "Update all" });
    expect(button).toBeDisabled();
  });

  it("shows Spinner while loading", () => {
    // Non-empty enhancedSoftware, isLoading
    const updates = createEnhancedSoftware(1);
    render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading // true
        isError={false}
      />
    );
    expect(screen.getByTestId("spinner")).toBeInTheDocument();
  });

  it("shows error view when isError is set", () => {
    const updates = createEnhancedSoftware(1);
    render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={updates}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError // true
      />
    );
    expect(
      screen.getByText(/This URL is invalid or expired./i)
    ).toBeInTheDocument();
  });

  it("renders nothing if there are no updatable apps", () => {
    // Software array, but all items have no update-needed ui_status
    const notUpdating = [
      {
        ...createMockDeviceSoftware({ id: 1 }),
        ui_status: "installed" as IHostSoftwareUiStatus,
      },
      {
        ...createMockDeviceSoftware({ id: 2 }),
        ui_status: "uninstall_failed" as IHostSoftwareUiStatus,
      },
    ];
    const { container } = render(
      <UpdatesCard
        contactUrl={contactUrl}
        enhancedSoftware={notUpdating}
        onClickUpdateAction={noop}
        onClickUpdateAll={noop}
        onClickFailedUpdateStatus={noop}
        isLoading={false}
        isError={false}
      />
    );
    expect(container.firstChild).toBeNull();
  });
  // Responsive sizing not tested
});
