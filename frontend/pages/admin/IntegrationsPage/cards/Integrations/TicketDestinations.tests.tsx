import {
  render as rtlRender,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import mockServer from "test/mock-server";
import { createCustomRenderer, baseUrl } from "test/test-utils";

import TicketDestinations from "./TicketDestinations";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const configUrl = baseUrl("/config");

const jiraIntegration = {
  url: "https://example.atlassian.net",
  username: "jira-user",
  api_token: "********",
  project_key: "PROJ",
  enable_failing_policies: false,
  enable_software_vulnerabilities: false,
};
const zendeskIntegration = {
  url: "https://example.zendesk.com",
  email: "zendesk-user@example.com",
  api_token: "********",
  group_id: 123,
  enable_failing_policies: false,
  enable_software_vulnerabilities: false,
};
const linearIntegration = { api_key: "********", team_id: "team-a" };

const createConfig = () =>
  createMockConfig({
    integrations: {
      jira: [jiraIntegration],
      zendesk: [zendeskIntegration],
      linear: [linearIntegration],
      google_calendar: [],
      google_workspace: [],
    } as IConfig["integrations"],
  });

const createGetConfigHandler = () =>
  http.get(configUrl, () => HttpResponse.json(createConfig()));

// Records the request body. Returns an error when `failure` is set.
const createPatchConfigHandler = (
  spy: jest.Mock,
  failure?: { status: number; body: object }
) =>
  http.patch(configUrl, async ({ request }) => {
    spy(await request.json());
    if (failure) {
      return HttpResponse.json(failure.body, { status: failure.status });
    }
    return HttpResponse.json(createConfig());
  });

// Toast messages are JSX. Render one to read its text.
const toastText = (toast: jest.Mock): string =>
  rtlRender(<>{toast.mock.calls[0][0]}</>).container.textContent || "";

const openAddModalWithLinear = async (user: UserEvent) => {
  await user.click(
    await screen.findByRole("button", { name: "Add integration" })
  );
  await user.click(screen.getByRole("combobox"));
  const linearOption = screen
    .getAllByTestId("dropdown-option")
    .find((option) => option.textContent?.startsWith("Linear"));
  if (!linearOption) {
    throw new Error('No "Linear" destination option');
  }
  await user.click(linearOption);
};

const submitLinearForm = async (user: UserEvent) => {
  await user.type(screen.getByLabelText("API key"), "lin_api_new");
  await user.type(screen.getByLabelText("Team ID"), "team-b");
  await user.click(screen.getByRole("button", { name: "Add" }));
};

const getRow = async (name: string) =>
  (await screen.findByText(name)).closest("tr") as HTMLElement;

describe("TicketDestinations", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("lists Jira, Zendesk, and Linear destinations with their icons", async () => {
    mockServer.use(createGetConfigHandler());
    render(<TicketDestinations />);

    const linearRow = await getRow("Linear - team-a");
    const jiraRow = await getRow("https://example.atlassian.net - PROJ");
    const zendeskRow = await getRow("https://example.zendesk.com - 123");

    // The Jest image transform returns the file name as the src.
    expect(within(linearRow).getByAltText("integration-icon")).toHaveAttribute(
      "src",
      "icon-linear-24x24@2x.png"
    );
    expect(within(jiraRow).getByAltText("integration-icon")).toHaveAttribute(
      "src",
      "icon-jira-24x24@2x.png"
    );
    expect(within(zendeskRow).getByAltText("integration-icon")).toHaveAttribute(
      "src",
      "icon-zendesk-32x24@2x.png"
    );
  });

  it("adds a Linear destination and keeps the existing ones", async () => {
    const patchSpy = jest.fn();
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(patchSpy)
    );
    const { user } = render(<TicketDestinations />);

    await openAddModalWithLinear(user);
    await submitLinearForm(user);

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith({
        integrations: {
          linear: [
            linearIntegration,
            { api_key: "lin_api_new", team_id: "team-b" },
          ],
        },
      });
    });
    await waitFor(() => {
      expect(notify.success).toHaveBeenCalledTimes(1);
    });
    expect(toastText(notify.success as jest.Mock)).toBe(
      "Successfully added Linear - team-b"
    );
  });

  it("shows a duplicate error when the Linear destination exists", async () => {
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(jest.fn(), {
        status: 422,
        body: {
          message: "Validation Failed",
          errors: [{ name: "base", reason: "duplicate Linear integration" }],
        },
      })
    );
    const { user } = render(<TicketDestinations />);

    await openAddModalWithLinear(user);
    await submitLinearForm(user);

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledTimes(1);
    });
    const text = toastText(notify.error as jest.Mock);
    expect(text).toContain("Could not add Linear - team-b");
    expect(text).toContain("This integration already exists");
    expect(notify.success).not.toHaveBeenCalled();
  });

  it("shows a generic error when adding a Linear destination fails", async () => {
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(jest.fn(), {
        status: 500,
        body: { message: "Something broke", errors: [] },
      })
    );
    const { user } = render(<TicketDestinations />);

    await openAddModalWithLinear(user);
    await submitLinearForm(user);

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledTimes(1);
    });
    expect(toastText(notify.error as jest.Mock)).toContain(
      "Could not add Linear"
    );
    expect(notify.success).not.toHaveBeenCalled();
  });

  it("shows an API key error when Linear rejects the credentials", async () => {
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(jest.fn(), {
        status: 400,
        body: { message: "Bad request: invalid key", errors: [] },
      })
    );
    const { user } = render(<TicketDestinations />);

    await openAddModalWithLinear(user);
    await submitLinearForm(user);

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledTimes(1);
    });
    expect(toastText(notify.error as jest.Mock)).toBe(
      "Invalid API key or team ID. Please correct and try again."
    );
    expect(notify.success).not.toHaveBeenCalled();
  });

  it("deletes a Linear destination and leaves Jira and Zendesk alone", async () => {
    const patchSpy = jest.fn();
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(patchSpy)
    );
    const { user } = render(<TicketDestinations />);

    const linearRow = await getRow("Linear - team-a");
    await user.click(
      within(linearRow).getByRole("button", { name: "Delete integration" })
    );

    expect(await screen.findByText("Delete integration")).toBeInTheDocument();
    // One match in the table row, one in the modal.
    expect(screen.getAllByText("Linear - team-a")).toHaveLength(2);

    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith({ integrations: { linear: [] } });
    });
    await waitFor(() => {
      expect(notify.success).toHaveBeenCalledTimes(1);
    });
    expect(toastText(notify.success as jest.Mock)).toBe(
      "Successfully deleted Linear - team-a"
    );
  });

  it("deletes a Jira destination and keeps the Zendesk list", async () => {
    const patchSpy = jest.fn();
    mockServer.use(
      createGetConfigHandler(),
      createPatchConfigHandler(patchSpy)
    );
    const { user } = render(<TicketDestinations />);

    const jiraRow = await getRow("https://example.atlassian.net - PROJ");
    await user.click(
      within(jiraRow).getByRole("button", { name: "Delete integration" })
    );
    await user.click(await screen.findByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith({
        integrations: { jira: [], zendesk: [zendeskIntegration] },
      });
    });
  });

  it("deletes from the latest list when the config reloads", async () => {
    const patchSpy = jest.fn();
    let reloaded = false;
    mockServer.use(
      http.get(configUrl, () => {
        const config = createConfig();
        if (reloaded) {
          config.integrations.linear = [
            linearIntegration,
            { api_key: "********", team_id: "team-c" },
          ];
        }
        return HttpResponse.json(config);
      }),
      createPatchConfigHandler(patchSpy)
    );
    const { user } = render(<TicketDestinations />);

    const linearRow = await getRow("Linear - team-a");
    await user.click(
      within(linearRow).getByRole("button", { name: "Delete integration" })
    );
    expect(await screen.findByText("Delete integration")).toBeInTheDocument();

    // Reload the config while the delete modal is open.
    reloaded = true;
    window.dispatchEvent(new Event("focus"));
    expect(await screen.findByText("Linear - team-c")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Delete" }));

    // A stale list would send `linear: []`.
    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith({
        integrations: {
          linear: [{ api_key: "********", team_id: "team-c" }],
        },
      });
    });
  });

  it("deletes the selected Linear destination when the list order changes", async () => {
    const patchSpy = jest.fn();
    const getSpy = jest.fn();
    const linearB = { api_key: "********", team_id: "team-b" };
    let reordered = false;
    mockServer.use(
      http.get(configUrl, () => {
        getSpy();
        const config = createConfig();
        config.integrations.linear = reordered
          ? [linearB, linearIntegration]
          : [linearIntegration, linearB];
        return HttpResponse.json(config);
      }),
      createPatchConfigHandler(patchSpy)
    );
    const { user } = render(<TicketDestinations />);

    const linearRow = await getRow("Linear - team-b");
    await user.click(
      within(linearRow).getByRole("button", { name: "Delete integration" })
    );
    expect(await screen.findByText("Delete integration")).toBeInTheDocument();

    // Reorder the list while the delete modal is open.
    reordered = true;
    window.dispatchEvent(new Event("focus"));
    await waitFor(() => {
      expect(getSpy).toHaveBeenCalledTimes(2);
    });

    await user.click(screen.getByRole("button", { name: "Delete" }));

    // An index filter would remove `team-a` here.
    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledWith({
        integrations: { linear: [linearIntegration] },
      });
    });
  });
});
