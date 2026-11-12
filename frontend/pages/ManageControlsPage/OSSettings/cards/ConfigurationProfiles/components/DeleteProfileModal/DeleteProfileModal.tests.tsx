import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { ProfilePlatform } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";

import DeleteProfileModal from "./DeleteProfileModal";

const render = createCustomRenderer();

const renderModal = (
  platform: ProfilePlatform,
  onDelete: (profileId: string) => void = noop
) =>
  render(
    <DeleteProfileModal
      profileName="Test profile"
      profileId="profile-uuid"
      platform={platform}
      onCancel={noop}
      onDelete={onDelete}
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

  it("doesn't delete when Enter is pressed on the Windows Learn more link", async () => {
    const onDelete = jest.fn();
    const { user } = renderModal("windows", onDelete);

    screen.getByRole("link", { name: /Learn more/i }).focus();
    await user.keyboard("{Enter}");

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("deletes on Enter for non-Windows profiles", async () => {
    const onDelete = jest.fn();
    const { user } = renderModal("darwin", onDelete);

    await user.keyboard("{Enter}");

    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(onDelete).toHaveBeenCalledWith("profile-uuid");
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
