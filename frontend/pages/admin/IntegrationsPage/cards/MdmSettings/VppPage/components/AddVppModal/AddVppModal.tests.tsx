import { screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import { IMdmVppToken } from "interfaces/mdm";
import {
  APP_CONTEXT_ALL_TEAMS_SUMMARY,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";
import mdmAppleAPI from "services/entities/mdm_apple";
import { createCustomRenderer } from "test/test-utils";

import AddVppModal from "./AddVppModal";

jest.mock("services/entities/mdm_apple");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const mockMdmAppleAPI = mdmAppleAPI as jest.Mocked<typeof mdmAppleAPI>;

const TOKEN: IMdmVppToken = {
  id: 7,
  org_name: "Acme K.K.",
  location: "Acme K.K.",
  country_code: "jp",
  renew_date: "2027-04-27T00:00:00Z",
  teams: null,
};

const renderModal = ({ gitOpsModeEnabled = false } = {}) => {
  const config = createMockConfig();
  config.gitops.gitops_mode_enabled = gitOpsModeEnabled;
  const render = createCustomRenderer({
    context: {
      app: {
        config,
        availableTeams: [
          APP_CONTEXT_ALL_TEAMS_SUMMARY,
          { id: 1, name: "Workstations" },
          { id: 2, name: "Mobile" },
          APP_CONTEXT_NO_TEAM_SUMMARY,
        ],
      },
    },
  });
  const onAdded = jest.fn();
  const onCancel = jest.fn();
  const result = render(<AddVppModal onAdded={onAdded} onCancel={onCancel} />);
  return { ...result, onAdded, onCancel };
};

const uploadToken = async (
  user: ReturnType<typeof renderModal>["user"],
  container: HTMLElement
) => {
  const input = container.ownerDocument.querySelector(
    "input[type='file']"
  ) as HTMLInputElement;
  await user.upload(
    input,
    new File(["token"], "token.vpptoken", { type: "application/octet-stream" })
  );
};

describe("AddVppModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockMdmAppleAPI.uploadVppToken.mockResolvedValue({ token: TOKEN });
    mockMdmAppleAPI.editVppTeams.mockResolvedValue({});
  });

  it("assigns the new token to all fleets by default", async () => {
    const { user, container, onAdded } = renderModal();

    expect(screen.getByRole("radio", { name: "All fleets" })).toBeChecked();
    await uploadToken(user, container);
    await user.click(screen.getByRole("button", { name: "Add VPP" }));

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(mockMdmAppleAPI.editVppTeams).toHaveBeenCalledWith({
      tokenId: TOKEN.id,
      teamIds: [],
    });
    expect(notify.success).toHaveBeenCalledWith("Added successfully.");
  });

  it("assigns the new token to the chosen fleets", async () => {
    const { user, container, onAdded } = renderModal();

    await uploadToken(user, container);
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    await user.click(screen.getByRole("checkbox", { name: "Mobile" }));
    await user.click(screen.getByRole("checkbox", { name: "Unassigned" }));
    await user.click(screen.getByRole("button", { name: "Add VPP" }));

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(mockMdmAppleAPI.editVppTeams).toHaveBeenCalledWith({
      tokenId: TOKEN.id,
      teamIds: [2, 0],
    });
  });

  it("explains that the token was added when assigning fleets fails", async () => {
    mockMdmAppleAPI.editVppTeams.mockRejectedValue(new Error("boom"));
    const { user, container, onAdded } = renderModal();

    await uploadToken(user, container);
    await user.click(screen.getByRole("button", { name: "Add VPP" }));

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(notify.error).toHaveBeenCalledWith(
      "VPP token added, but couldn't assign fleets. Please edit fleets and try again.",
      expect.anything()
    );
    expect(notify.success).not.toHaveBeenCalled();
  });

  it("hides the fleet picker and leaves the token unassigned in GitOps mode", async () => {
    const { user, container, onAdded } = renderModal({
      gitOpsModeEnabled: true,
    });

    expect(screen.queryByRole("radio", { name: "All fleets" })).toBeNull();
    await uploadToken(user, container);
    await user.click(screen.getByRole("button", { name: "Add VPP" }));

    await waitFor(() => expect(onAdded).toHaveBeenCalled());
    expect(mockMdmAppleAPI.editVppTeams).not.toHaveBeenCalled();
  });
});
