import React from "react";
import { screen, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import createMockQuery from "__mocks__/queryMock";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import SaveQueryModal from "./SaveQueryModal";

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
  };

  it("renders the modal with initial values and allows editing", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
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
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(<SaveQueryModal {...defaultProps} />);

    const advancedOptionsButton = screen.getByText("Show advanced options");
    await user.click(advancedOptionsButton);

    expect(screen.getByText("Platforms")).toBeInTheDocument();
    expect(screen.getByText("Minimum osquery version")).toBeInTheDocument();
    expect(screen.getByText("Logging")).toBeInTheDocument();

    await user.click(advancedOptionsButton);

    expect(screen.queryByText("Platforms")).not.toBeInTheDocument();
  });

  it("displays error when query name is empty", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(<SaveQueryModal {...defaultProps} />);

    await user.click(screen.getByText("Save"));

    expect(screen.getByText("Query name must be present")).toBeInTheDocument();
  });
});
