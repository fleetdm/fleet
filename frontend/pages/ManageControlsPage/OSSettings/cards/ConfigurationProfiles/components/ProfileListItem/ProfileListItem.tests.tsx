import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { IMdmProfile } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";

import ProfileListItem from "./ProfileListItem";

const render = createCustomRenderer();

const baseProfile: IMdmProfile = {
  profile_uuid: "d123",
  team_id: 0,
  name: "My DDM profile",
  platform: "darwin",
  identifier: "com.example.test",
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  checksum: null,
  self_service: false,
  hidden: false,
};

const renderItem = (profile: IMdmProfile) =>
  render(
    <ProfileListItem
      isPremium={false}
      profile={profile}
      onClickInfo={noop}
      onClickEdit={noop}
      onClickDelete={noop}
    />
  );

describe("ProfileListItem", () => {
  it("shows the user-scope indicator for user-scoped profiles", () => {
    renderItem({ ...baseProfile, scope: "User" });
    expect(screen.getByTestId("user-icon")).toBeInTheDocument();
  });

  it("does not show the indicator for system-scoped profiles", () => {
    renderItem({ ...baseProfile, scope: "System" });
    expect(screen.queryByTestId("user-icon")).not.toBeInTheDocument();
  });

  it("does not show the indicator for iOS/iPadOS profiles (no user channel)", () => {
    renderItem({ ...baseProfile, platform: "ios", scope: "User" });
    expect(screen.queryByTestId("user-icon")).not.toBeInTheDocument();
  });

  it("shows a renamed mobileconfig's PayloadDisplayName and UUID on hover", async () => {
    const { user } = renderItem({
      ...baseProfile,
      name: "Renamed profile",
      payload_display_name: "Original display name",
    });

    await user.hover(screen.getByText("Renamed profile"));

    await waitFor(() => {
      expect(screen.getByText(/PayloadDisplayName:/)).toBeInTheDocument();
    });
    expect(screen.getByText("Original display name")).toBeInTheDocument();
    expect(screen.getByText("d123")).toBeInTheDocument();
  });

  it("shows only the UUID on hover when there's no PayloadDisplayName", async () => {
    const { user } = renderItem(baseProfile);

    await user.hover(screen.getByText("My DDM profile"));

    await waitFor(() => {
      expect(screen.getByText("d123")).toBeInTheDocument();
    });
    expect(screen.queryByText(/PayloadDisplayName:/)).not.toBeInTheDocument();
  });
});
