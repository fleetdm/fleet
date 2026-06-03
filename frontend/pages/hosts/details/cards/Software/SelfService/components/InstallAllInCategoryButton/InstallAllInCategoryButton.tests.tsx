import React from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";

import { createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { baseUrl } from "test/default-handlers";

import InstallAllInCategoryButton, {
  IInstallAllInCategoryButtonProps,
} from "./InstallAllInCategoryButton";

const baseProps: IInstallAllInCategoryButtonProps = {
  uninstalledCount: 3,
  hasInProgressInCategory: false,
  deviceToken: "test-token",
  categoryId: 1,
  onSuccess: jest.fn(),
};

describe("InstallAllInCategoryButton", () => {
  it("renders the uninstalled count in the label", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<InstallAllInCategoryButton {...baseProps} />);
    expect(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    ).toBeInTheDocument();
  });

  it("is disabled when uninstalledCount is 0", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<InstallAllInCategoryButton {...baseProps} uninstalledCount={0} />);
    expect(screen.getByRole("button", { name: /Install all/i })).toBeDisabled();
  });

  it("is disabled when hasInProgressInCategory is true", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <InstallAllInCategoryButton {...baseProps} hasInProgressInCategory />
    );
    expect(screen.getByRole("button", { name: /Install all/i })).toBeDisabled();
  });

  it("opens the confirmation modal when clicked", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(<InstallAllInCategoryButton {...baseProps} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    );

    expect(
      await screen.findByText(/3 new apps will be installed/i)
    ).toBeInTheDocument();
  });

  it("uses singular 'app' in the modal copy when count is 1", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(<InstallAllInCategoryButton {...baseProps} uninstalledCount={1} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(1\)/i })
    );

    expect(
      await screen.findByText(/1 new app will be installed/i)
    ).toBeInTheDocument();
  });

  it("closes the modal without POSTing when Cancel is clicked", async () => {
    let postCalled = false;
    mockServer.use(
      http.post(baseUrl("/device/:token/software/install_all"), () => {
        postCalled = true;
        return new HttpResponse(null, { status: 202 });
      })
    );
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(<InstallAllInCategoryButton {...baseProps} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    );
    await user.click(await screen.findByRole("button", { name: /Cancel/i }));

    await waitFor(() => {
      expect(
        screen.queryByText(/new apps will be installed/i)
      ).not.toBeInTheDocument();
    });
    expect(postCalled).toBe(false);
  });

  it("POSTs with the category_id query param when Confirm is clicked", async () => {
    let requestedUrl = "";
    mockServer.use(
      http.post(
        baseUrl("/device/:token/software/install_all"),
        ({ request }) => {
          requestedUrl = request.url;
          return new HttpResponse(null, { status: 202 });
        }
      )
    );
    const onSuccess = jest.fn();
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(<InstallAllInCategoryButton {...baseProps} onSuccess={onSuccess} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    );
    await user.click(
      await screen.findByRole("button", { name: /^Install all$/i })
    );

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    });
    expect(requestedUrl).toContain("category_id=1");
  });

  it("omits category_id when categoryId is undefined ('All' selected)", async () => {
    let requestedUrl = "";
    mockServer.use(
      http.post(
        baseUrl("/device/:token/software/install_all"),
        ({ request }) => {
          requestedUrl = request.url;
          return new HttpResponse(null, { status: 202 });
        }
      )
    );
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(
      <InstallAllInCategoryButton {...baseProps} categoryId={undefined} />
    );

    await user.click(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    );
    await user.click(
      await screen.findByRole("button", { name: /^Install all$/i })
    );

    await waitFor(() => {
      expect(requestedUrl).not.toContain("category_id");
    });
  });

  it("does not fire onSuccess and keeps the modal open when the install_all request fails", async () => {
    mockServer.use(
      http.post(baseUrl("/device/:token/software/install_all"), () =>
        HttpResponse.json(
          { errors: [{ name: "base", reason: "boom" }] },
          { status: 500 }
        )
      )
    );
    const onSuccess = jest.fn();
    const render = createCustomRenderer({ withBackendMock: true });
    const user = userEvent.setup();
    render(<InstallAllInCategoryButton {...baseProps} onSuccess={onSuccess} />);

    await user.click(
      screen.getByRole("button", { name: /Install all \(3\)/i })
    );
    await user.click(
      await screen.findByRole("button", { name: /^Install all$/i })
    );

    // Give the failed promise a tick to settle.
    await waitFor(() => {
      expect(onSuccess).not.toHaveBeenCalled();
    });
    // Modal stays open so the user can retry; only the success path closes it.
    expect(
      screen.getByText(/3 new apps will be installed/i)
    ).toBeInTheDocument();
  });
});
