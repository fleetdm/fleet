import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createMockVppToken } from "__mocks__/mdmMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import AddVppModal from "./AddVppModal";

jest.mock("components/ToastNotification", () => ({
  notify: { success: jest.fn(), error: jest.fn(), batch: jest.fn() },
}));

const availableTeams = [
  createMockTeamSummary({ id: -1, name: "All fleets" }),
  createMockTeamSummary({ id: 1, name: "Workstations" }),
  createMockTeamSummary({ id: 2, name: "Personal mobile devices" }),
  createMockTeamSummary({ id: 0, name: "Unassigned" }),
];

const newToken = createMockVppToken({ id: 9 });

interface IMockOptions {
  patchStatus?: number;
}

const mockEndpoints = ({ patchStatus = 200 }: IMockOptions = {}) => {
  const calls = { posts: 0, patches: [] as unknown[] };
  mockServer.use(
    http.post(baseUrl("/vpp_tokens"), () => {
      calls.posts += 1;
      return HttpResponse.json({ token: newToken }, { status: 202 });
    }),
    http.patch(baseUrl("/vpp_tokens/9/fleets"), async ({ request }) => {
      calls.patches.push(await request.json());
      return patchStatus === 200
        ? HttpResponse.json({})
        : HttpResponse.json({ message: "boom" }, { status: patchStatus });
    })
  );
  return calls;
};

const renderModal = (gitOpsModeEnabled = false) => {
  const onAdded = jest.fn();
  const onCancel = jest.fn();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        availableTeams,
        config: createMockConfig({
          gitops: {
            gitops_mode_enabled: gitOpsModeEnabled,
            repository_url: "https://example.com/repo",
            exceptions: { labels: false, software: false, secrets: false },
          },
        }),
      },
    },
  });
  const utils = render(<AddVppModal onAdded={onAdded} onCancel={onCancel} />);
  return { ...utils, onAdded, onCancel };
};

const upload = async (user: ReturnType<typeof renderModal>["user"]) => {
  await user.upload(
    document.querySelector("input[type=file]") as HTMLInputElement,
    new File(["token"], "acme.vpptoken")
  );
  await user.click(screen.getByRole("button", { name: "Add VPP" }));
};

describe("AddVppModal", () => {
  beforeEach(() => jest.clearAllMocks());

  it("links to the guide with 'Learn more'", () => {
    renderModal();

    expect(screen.getAllByText("Learn more").length).toBeGreaterThan(0);
    expect(screen.queryByText("Learn how")).toBeNull();
  });

  it("links to the token best practices from the Fleets tooltip", async () => {
    const { user } = renderModal();

    await user.hover(screen.getByText("Fleets"));

    const tooltip = await screen.findByRole("tooltip");
    expect(
      within(tooltip).getByRole("link", { name: /Learn more/ })
    ).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/vpp-tokens-best-practice"
    );
  });

  it("disables the fleet picker while the upload is in flight", async () => {
    let finishPost!: () => void;
    const postPending = new Promise<void>((resolve) => {
      finishPost = resolve;
    });
    mockServer.use(
      http.post(baseUrl("/vpp_tokens"), async () => {
        await postPending;
        return HttpResponse.json({ token: newToken }, { status: 202 });
      }),
      http.patch(baseUrl("/vpp_tokens/9/fleets"), () => HttpResponse.json({}))
    );
    const { user, onAdded } = renderModal();

    await upload(user);

    expect(screen.getByRole("radio", { name: "All fleets" })).toBeDisabled();
    expect(screen.getByRole("radio", { name: "Custom" })).toBeDisabled();

    finishPost();
    await waitFor(() => expect(onAdded).toHaveBeenCalled());
  });

  it("defaults to All fleets and sends POST then PATCH []", async () => {
    const calls = mockEndpoints();
    const { user, onAdded } = renderModal();

    expect(screen.getByRole("radio", { name: "All fleets" })).toBeChecked();
    await upload(user);

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(calls.posts).toBe(1);
    expect(calls.patches).toEqual([{ fleets: [] }]);
    expect(notify.success).toHaveBeenCalledWith("Added successfully.");
  });

  it("sends the chosen fleet ids for Custom", async () => {
    const calls = mockEndpoints();
    const { user, onAdded } = renderModal();

    await user.click(screen.getByRole("radio", { name: "Custom" }));
    await user.click(screen.getByRole("checkbox", { name: "Workstations" }));
    await user.click(screen.getByRole("checkbox", { name: "Unassigned" }));
    await upload(user);

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(calls.patches).toEqual([{ fleets: [1, 0] }]);
  });

  it("sends only the POST for Custom with no fleets", async () => {
    const calls = mockEndpoints();
    const { user, onAdded } = renderModal();

    await user.click(screen.getByRole("radio", { name: "Custom" }));
    await upload(user);

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(calls.posts).toBe(1);
    expect(calls.patches).toEqual([]);
  });

  it("toasts and refetches when the PATCH fails after the POST", async () => {
    const calls = mockEndpoints({ patchStatus: 500 });
    const { user, onAdded, onCancel } = renderModal();

    await upload(user);

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(calls.posts).toBe(1);
    expect(notify.error).toHaveBeenCalledWith(
      expect.stringContaining("The VPP token was added"),
      expect.anything()
    );
    expect(notify.success).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("cancels and shows the error when the POST fails", async () => {
    mockServer.use(
      http.post(baseUrl("/vpp_tokens"), () =>
        HttpResponse.json({ message: "bad" }, { status: 500 })
      )
    );
    const { user, onAdded, onCancel } = renderModal();

    await upload(user);

    await waitFor(() => expect(onCancel).toHaveBeenCalled());
    expect(onAdded).not.toHaveBeenCalled();
    expect(notify.error).toHaveBeenCalled();
  });

  it("hides the picker and skips the PATCH in GitOps mode", async () => {
    const calls = mockEndpoints();
    const { user, onAdded } = renderModal(true);

    expect(screen.queryByRole("radio", { name: "All fleets" })).toBeNull();
    await upload(user);

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(calls.posts).toBe(1);
    expect(calls.patches).toEqual([]);
  });
});
