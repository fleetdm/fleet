import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import DeleteHostModal from "./DeleteHostModal";

const renderModal = (
  props: Partial<React.ComponentProps<typeof DeleteHostModal>>,
  useOneTimeEnrollSecrets = false
) => {
  const render = createCustomRenderer({
    context: {
      app: {
        config: createMockConfig({
          auth: { use_one_time_enroll_secrets: useOneTimeEnrollSecrets },
        }),
      },
    },
  });
  return render(
    <DeleteHostModal
      onSubmit={noop}
      onCancel={noop}
      isUpdating={false}
      {...props}
    />
  );
};

const DELETING_A_HOST_LINK = /learn-more-about\/deleting-a-host$/;

describe("DeleteHostModal", () => {
  it("renders the number of hosts selected", () => {
    renderModal({ selectedHostIds: [1, 2, 3] });
    expect(screen.getByText("3 hosts")).toBeVisible();
  });

  it("renders the host name when only the host name is provided", () => {
    renderModal({ hostName: "Host1" });
    expect(screen.getByText("Host1")).toBeVisible();
  });

  it("renders the total hosts count when select all matching hosts is true", () => {
    renderModal({
      selectedHostIds: [1, 2, 3],
      hostsCount: 50,
      isAllMatchingHostsSelected: true,
    });
    expect(screen.getByText("50 hosts")).toBeVisible();
  });

  it("renders the host count with an additional warning when there are more than 500 hosts and select all matching hosts is true", () => {
    renderModal({
      selectedHostIds: [1, 2, 3],
      hostsCount: 500,
      isAllMatchingHostsSelected: true,
    });
    expect(screen.getByText("500 hosts")).toBeVisible();
    expect(
      screen.getByText(
        "When deleting a large volume of hosts, it may take some time for this change to be reflected in the UI."
      )
    ).toBeVisible();
  });

  it("renders the generic copy for a macOS host that is not enrolled in Fleet MDM", () => {
    renderModal({ hostName: "Host1", platform: "darwin" });
    expect(
      screen.getByText(/and associated data such as unlock PINs/i)
    ).toBeVisible();
    expect(
      screen.getByText(
        /iOS and iPadOS will re-enroll unless MDM is turned off/i
      )
    ).toBeVisible();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      expect.stringMatching(DELETING_A_HOST_LINK)
    );
  });

  it("renders the Android copy", () => {
    renderModal({ hostName: "Pixel", platform: "android" });
    expect(screen.getByText("Pixel")).toBeVisible();
    expect(screen.getByText(/and remove company data\./i)).toBeVisible();
    expect(screen.getByText(/This may take up to 24 hours\./)).toBeVisible();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      expect.stringMatching(DELETING_A_HOST_LINK)
    );
  });

  it("renders the iOS and iPadOS copy", () => {
    renderModal({ hostName: "iPad", platform: "ipados" });
    expect(screen.getByText("This will remove all host data.")).toBeVisible();
    expect(
      screen.getByText(/This host will re-enroll unless MDM is turned off\./)
    ).toBeVisible();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      expect.stringMatching(DELETING_A_HOST_LINK)
    );
  });

  it("renders the macOS MDM copy with the deleting-a-host link when one-time enroll secrets are off", () => {
    renderModal({
      hostName: "Mac",
      platform: "darwin",
      isMdmEnrolledInFleet: true,
    });
    expect(
      screen.getByText(
        "This will remove all host data such as unlock PINs and disk encryption keys."
      )
    ).toBeVisible();
    expect(
      screen.getByText(/re-enroll unless Fleet's agent is uninstalled\./i)
    ).toBeVisible();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      expect.stringMatching(DELETING_A_HOST_LINK)
    );
  });

  it("tells admins to reinstall the agent for a manually enrolled Mac when one-time enroll secrets are on", () => {
    renderModal(
      {
        hostName: "Mac",
        platform: "darwin",
        isMdmEnrolledInFleet: true,
        mdmEnrollmentStatus: "On (manual)",
      },
      true
    );
    expect(screen.getByText("Mac")).toBeVisible();
    expect(screen.getByText(/but won't remove company data\./i)).toBeVisible();
    expect(
      screen.getByText("To re-enroll it, Fleet's agent must be reinstalled.")
    ).toBeVisible();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("renders the profiles renew instructions for an automatically enrolled Mac when one-time enroll secrets are on", () => {
    renderModal(
      {
        hostName: "Mac",
        platform: "darwin",
        isMdmEnrolledInFleet: true,
        mdmEnrollmentStatus: "On (automatic)",
      },
      true
    );
    expect(screen.getByText("Mac")).toBeVisible();
    expect(screen.getByText(/but won't remove company data\./i)).toBeVisible();
    expect(screen.getByText("profiles renew -type enrollment")).toBeVisible();
    expect(screen.getByRole("link", { name: /learn more/i })).toHaveAttribute(
      "href",
      expect.stringMatching(DELETING_A_HOST_LINK)
    );
  });
});
