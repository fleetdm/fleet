import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { ProfilePlatform } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";

import DeleteProfileModal from "./DeleteProfileModal";

const render = createCustomRenderer();

const renderModal = (platform: ProfilePlatform) =>
  render(
    <DeleteProfileModal
      profileName="Test profile"
      profileId="profile-uuid"
      platform={platform}
      onCancel={noop}
      onDelete={noop}
      isDeleting={false}
    />
  );

describe("DeleteProfileModal", () => {
  it("warns that Windows settings may not be cleanly removed", () => {
    renderModal("windows");

    expect(
      screen.getByText(
        /Some Windows settings may not be cleanly removed from hosts/i
      )
    ).toBeInTheDocument();
    const link = screen.getByRole("link", { name: /Learn more/i });
    expect(link).toHaveAttribute(
      "href",
      "https://fleetdm.com/learn-more-about/removal-behavior"
    );
    expect(link).toHaveAttribute("target", "_blank");
  });

  it.each<ProfilePlatform>(["darwin", "ios", "ipados", "android", "linux"])(
    "doesn't show the Windows warning for %s profiles",
    (platform) => {
      renderModal(platform);

      expect(
        screen.getByText(/Pending profiles will be canceled/i)
      ).toBeInTheDocument();
      expect(
        screen.queryByText(/Some Windows settings/i)
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("link", { name: /Learn more/i })
      ).not.toBeInTheDocument();
    }
  );
});
