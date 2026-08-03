import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";

import {
  createMockFleetMaintainedAppDetails,
  createMockSoftwarePackage,
  createMockSoftwareTitle,
} from "__mocks__/softwareMock";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";
import { createCustomRenderer } from "test/test-utils";

import DeployModal from "./DeployModal";

const renderModal = ({
  softwarePackage = createMockSoftwarePackage({ fleet_maintained_app_id: 1 }),
  gitOpsModeEnabled = false,
  onExit = noop,
  onSuccess = noop,
}: {
  softwarePackage?: ReturnType<typeof createMockSoftwarePackage>;
  gitOpsModeEnabled?: boolean;
  onExit?: () => void;
  onSuccess?: () => void;
} = {}) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: {
          gitops: { gitops_mode_enabled: gitOpsModeEnabled },
        },
      },
    },
  });
  return render(
    <DeployModal
      softwareTitle={createMockSoftwareTitle({
        id: 10,
        name: "Firefox",
        software_package: softwarePackage,
      })}
      teamId={1}
      onExit={onExit}
      onSuccess={onSuccess}
    />
  );
};

describe("DeployModal", () => {
  beforeEach(() => {
    jest.spyOn(softwareAPI, "getFleetMaintainedApp").mockResolvedValue({
      fleet_maintained_app: createMockFleetMaintainedAppDetails(),
    });
    jest.spyOn(teamPoliciesAPI, "create").mockResolvedValue({} as never);
    jest.spyOn(teamPoliciesAPI, "update").mockResolvedValue({} as never);
    jest.spyOn(teamPoliciesAPI, "destroy").mockResolvedValue({} as never);
  });

  afterEach(() => jest.restoreAllMocks());

  it("reflects an externally-created manual patch policy", () => {
    renderModal({
      softwarePackage: createMockSoftwarePackage({
        fleet_maintained_app_id: 1,
        patch_policy: {
          id: 22,
          name: "Firefox up to date",
          patch_when_closed: false,
        },
        automatic_install_policies: [],
      }),
    });

    expect(screen.getByRole("checkbox", { name: "patch" })).toBeChecked();
    expect(
      screen.getByRole("radio", { name: "End user initiated (manual)" })
    ).toBeChecked();
  });

  it("creates a Force patch policy through the policy endpoint", async () => {
    const { user } = renderModal();

    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("radio", { name: "Force patch" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(teamPoliciesAPI.create).toHaveBeenCalledWith({
        team_id: 1,
        type: "patch",
        patch_software_title_id: 10,
        software_title_id: 10,
        patch_when_closed: false,
        continuous_automations_enabled: true,
      })
    );
  });

  it("creates Force install with the current FMA query and platform", async () => {
    const { user } = renderModal();

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled()
    );
    await user.click(screen.getByRole("checkbox", { name: "force-install" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(teamPoliciesAPI.create).toHaveBeenCalledWith({
        team_id: 1,
        name: "[Install software] Firefox",
        description:
          "Policy triggers automatic install of Firefox on each host that's missing this software.",
        query:
          "SELECT 1 FROM apps WHERE bundle_identifier = 'com.example.test-app';",
        platform: "darwin",
        software_title_id: 10,
      })
    );
  });

  it("attaches install automation when a manual patch policy changes to Force patch", async () => {
    const { user } = renderModal({
      softwarePackage: createMockSoftwarePackage({
        fleet_maintained_app_id: 1,
        patch_policy: {
          id: 22,
          name: "Firefox up to date",
          patch_when_closed: false,
        },
        automatic_install_policies: [],
      }),
    });

    await user.click(screen.getByRole("radio", { name: "Force patch" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(teamPoliciesAPI.update).toHaveBeenCalledWith(22, {
        team_id: 1,
        software_title_id: 10,
        patch_when_closed: false,
        continuous_automations_enabled: true,
      })
    );
  });

  it("removes install automation when Force patch changes to manual", async () => {
    const { user } = renderModal({
      softwarePackage: createMockSoftwarePackage({
        fleet_maintained_app_id: 1,
        patch_policy: {
          id: 22,
          name: "Firefox up to date",
          patch_when_closed: false,
        },
        automatic_install_policies: [
          { id: 22, name: "Firefox up to date", type: "patch" },
        ],
      }),
    });

    await user.click(
      screen.getByRole("radio", { name: "End user initiated (manual)" })
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(teamPoliciesAPI.update).toHaveBeenCalledWith(22, {
        team_id: 1,
        software_title_id: null,
        patch_when_closed: false,
        continuous_automations_enabled: false,
      })
    );
  });

  it("enables continuous automation when saving a migrated Force patch policy", async () => {
    const { user } = renderModal({
      softwarePackage: createMockSoftwarePackage({
        fleet_maintained_app_id: 1,
        patch_policy: {
          id: 22,
          name: "Firefox up to date",
          patch_when_closed: false,
          continuous_automations_enabled: false,
        },
        automatic_install_policies: [
          { id: 22, name: "Firefox up to date", type: "patch" },
        ],
      }),
    });

    expect(screen.getByRole("radio", { name: "Force patch" })).toBeChecked();
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(teamPoliciesAPI.update).toHaveBeenCalledWith(22, {
        team_id: 1,
        software_title_id: 10,
        patch_when_closed: false,
        continuous_automations_enabled: true,
      })
    );
  });

  it("deletes Force install and Patch policies independently", async () => {
    const { user } = renderModal({
      softwarePackage: createMockSoftwarePackage({
        fleet_maintained_app_id: 1,
        automatic_install_policies: [
          { id: 11, name: "[Install software] Firefox", type: "dynamic" },
        ],
        patch_policy: {
          id: 22,
          name: "Firefox up to date",
          patch_when_closed: true,
        },
      }),
    });

    await user.click(screen.getByRole("checkbox", { name: "force-install" }));
    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(teamPoliciesAPI.destroy).toHaveBeenNthCalledWith(1, 1, [11]);
      expect(teamPoliciesAPI.destroy).toHaveBeenNthCalledWith(2, 1, [22]);
    });
  });

  it("closes and refreshes after a partial save so retry uses fresh policy state", async () => {
    const onExit = jest.fn();
    const onSuccess = jest.fn();
    (teamPoliciesAPI.create as jest.Mock)
      .mockResolvedValueOnce({})
      .mockRejectedValueOnce(new Error("Patch failed"));
    const { user } = renderModal({ onExit, onSuccess });

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Save" })).toBeEnabled()
    );
    await user.click(screen.getByRole("checkbox", { name: "force-install" }));
    await user.click(screen.getByRole("checkbox", { name: "patch" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(teamPoliciesAPI.create).toHaveBeenCalledTimes(2);
      expect(onSuccess).toHaveBeenCalledTimes(1);
      expect(onExit).toHaveBeenCalledTimes(1);
    });
  });

  it("disables the Deploy control and Save in GitOps mode", () => {
    renderModal({ gitOpsModeEnabled: true });

    expect(
      screen.getByRole("checkbox", { name: "force-install" })
    ).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByRole("checkbox", { name: "patch" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
  });
});
