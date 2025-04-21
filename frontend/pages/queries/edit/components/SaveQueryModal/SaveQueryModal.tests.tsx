import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import { QueryablePlatform } from "interfaces/platform";

import SaveQueryModal from "./SaveQueryModal";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const mockLabels = [
  {
    id: 1,
    name: "Fun",
    description: "Computers that like to have a good time",
    label_type: "regular",
  },
  {
    id: 2,
    name: "Fresh",
    description: "Laptops with dirty mouths",
    label_type: "regular",
  },
];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({
    labels: mockLabels,
  });
});

const mockQuery = createMockQuery();

describe("SaveQueryModal", () => {
  const defaultProps = {
    queryValue: "SELECT * FROM users",
    apiTeamIdForQuery: 1,
    isLoading: false,
    saveQuery: jest.fn(),
    toggleSaveQueryModal: jest.fn(),
    backendValidators: {},
    existingQuery: mockQuery,
    queryReportsDisabled: false,
    platformSelector: {
      getSelectedPlatforms: () => ["linux"] as QueryablePlatform[],
      setSelectedPlatforms: jest.fn(),
      isAnyPlatformSelected: true,
      render: () => <></>,
    },
  };

  it("renders the modal with initial values and allows editing", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    const { user } = render(<SaveQueryModal {...defaultProps} />);

    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Description")).toBeInTheDocument();
    expect(screen.getByText("Frequency")).toBeInTheDocument();
    expect(screen.getByText("Observers can run")).toBeInTheDocument();
    expect(screen.getByText("Automations off")).toBeInTheDocument();
    expect(screen.getByText("Show advanced options")).toBeInTheDocument();

    const nameInput = screen.getByLabelText("Name");
    await user.type(nameInput, "Test Query");
    expect(nameInput).toHaveValue("Test Query");
  });

  it("toggles advanced options", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    const { user } = render(<SaveQueryModal {...defaultProps} />);

    const advancedOptionsButton = screen.getByText("Show advanced options");
    await user.click(advancedOptionsButton);

    expect(screen.getByText("Minimum osquery version")).toBeInTheDocument();
    expect(screen.getByText("Logging")).toBeInTheDocument();

    await user.click(advancedOptionsButton);
  });

  it("displays error when query name is empty", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    const { user } = render(<SaveQueryModal {...defaultProps} />);

    await user.click(screen.getByText("Save"));

    expect(screen.getByText("Query name must be present")).toBeInTheDocument();
  });

  it("should not show the target selector in the free tier", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
          isPremiumTier: false,
        },
      },
    });

    render(<SaveQueryModal {...defaultProps} />);

    // Wait for any queries (that should not be happening) to finish.
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Check that the target selector is not present.
    expect(screen.queryByText("All hosts")).not.toBeInTheDocument();
  });

  it("should disable the save button in when no platforms are selected", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: false,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });

    const props = {
      ...defaultProps,
      platformSelector: {
        getSelectedPlatforms: () => [] as QueryablePlatform[],
        setSelectedPlatforms: jest.fn(),
        isAnyPlatformSelected: false,
        render: () => <></>,
      },
    };

    render(<SaveQueryModal {...props} />);
    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();
  });

  it("should send platforms when saving a new query", async () => {
    const saveQuery = jest.fn();
    const props = {
      ...defaultProps,
      platformSelector: {
        getSelectedPlatforms: () => ["linux", "macos"] as QueryablePlatform[],
        setSelectedPlatforms: jest.fn(),
        isAnyPlatformSelected: true,
        render: () => <></>,
      },
      saveQuery,
    };
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: false,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });
    render(<SaveQueryModal {...props} />);
    await waitFor(() => {
      expect(screen.getByLabelText("Name")).toBeInTheDocument();
    });
    // Set a name.
    await userEvent.type(screen.getByLabelText("Name"), "A Brand New Query!");
    // Set a label.
    await userEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(saveQuery.mock.calls[0][0].platform).toEqual("linux,macos");
  });

  describe("in premium tier", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          currentUser: createMockUser(),
          isGlobalObserver: false,
          isGlobalAdmin: true,
          isGlobalMaintainer: false,
          isOnGlobalTeam: true,
          isPremiumTier: true,
          isSandboxMode: false,
          config: createMockConfig(),
        },
      },
    });

    beforeEach(() => {
      mockServer.use(labelSummariesHandler);
    });

    it("should show the target selector in All hosts target mode when the query has no labels", async () => {
      render(<SaveQueryModal {...defaultProps} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
        expect(screen.getByLabelText("Custom")).toBeInTheDocument();
        expect(screen.getByLabelText("All hosts")).toBeChecked();
      });
    });

    it("should disable the save button in Custom target mode when no labels are selected, and enable it once labels are selected", async () => {
      render(<SaveQueryModal {...defaultProps} />);
      let allHosts;
      let custom;
      await waitFor(() => {
        allHosts = screen.getByLabelText("All hosts");
        custom = screen.getByLabelText("Custom");
        expect(allHosts).toBeInTheDocument();
        expect(custom).toBeInTheDocument();
      });
      custom && (await userEvent.click(custom));
      const saveButton = screen.getByRole("button", { name: "Save" });
      expect(saveButton).toBeDisabled();

      const funButton = screen.getByLabelText("Fun");
      expect(funButton).not.toBeChecked();
      await userEvent.click(funButton);
      expect(saveButton).toBeEnabled();
    });

    it("should send labels when saving a new query in Custom target mode", async () => {
      const saveQuery = jest.fn();
      const props = { ...defaultProps, saveQuery };
      render(<SaveQueryModal {...props} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      // Set a name.
      await userEvent.type(screen.getByLabelText("Name"), "A Brand New Query!");

      // Set a label.
      await userEvent.click(screen.getByLabelText("Custom"));
      await userEvent.click(screen.getByLabelText("Fun"));
      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(saveQuery.mock.calls[0][0].labels_include_any).toEqual(["Fun"]);
    });

    it("should clear labels when saving a new query in All hosts target mode", async () => {
      const saveQuery = jest.fn();
      const props = { ...defaultProps, saveQuery };
      render(<SaveQueryModal {...props} />);
      await waitFor(() => {
        expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      });

      // Set a name.
      await userEvent.type(screen.getByLabelText("Name"), "A Brand New Query!");

      await userEvent.click(screen.getByRole("button", { name: "Save" }));

      expect(saveQuery.mock.calls[0][0].labels_include_any).toEqual([]);
    });
  });
});
