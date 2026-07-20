import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";
import { createCustomRenderer } from "test/test-utils";
import configAPI from "services/entities/config";

import AddEntraClientIdModal from "./AddEntraClientIDModal";

jest.mock("services/entities/config");

// A valid (version 4) UUID stored in upper-case, as it might be after being added via GitOps or the API.
const EXISTING_UPPERCASE_ID = "6D8769E6-0F8B-418D-B385-1A53968781C9";

const createTestMockData = (configOverrides: Partial<IConfig>) => ({
  context: {
    app: {
      isPremiumTier: true,
      config: createMockConfig(configOverrides),
      setConfig: jest.fn(),
    },
    notification: {
      renderFlash: jest.fn(),
    },
  },
});

describe("AddEntraClientIdModal", () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  it("rejects a case-insensitive duplicate of an existing client ID without calling the API", async () => {
    const renderFlash = jest.fn();
    const mockData = createTestMockData({
      mdm: createMockMdmConfig({
        windows_entra_client_ids: [EXISTING_UPPERCASE_ID],
      }),
    });
    mockData.context.notification.renderFlash = renderFlash;

    const render = createCustomRenderer(mockData);
    const { user } = render(<AddEntraClientIdModal onExit={jest.fn()} />);

    // Enter the same GUID in lower-case: it differs only in case from the stored upper-case entry, so it
    // must be treated as a duplicate (the backend authorizes client IDs case-insensitively).
    const input = screen.getByRole("textbox", { name: "Client ID" });
    await user.type(input, EXISTING_UPPERCASE_ID.toLowerCase());
    await user.click(screen.getByRole("button", { name: "Add" }));

    await waitFor(() => {
      expect(renderFlash).toHaveBeenCalledWith(
        "error",
        "Couldn't add client ID. Client ID already exists."
      );
    });
    expect(configAPI.update).not.toHaveBeenCalled();
  });
});
