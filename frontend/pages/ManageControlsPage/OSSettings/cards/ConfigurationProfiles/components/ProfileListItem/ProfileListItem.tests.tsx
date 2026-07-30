import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";

import { createCustomRenderer } from "test/test-utils";
import { IMdmProfile } from "interfaces/mdm";

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
});
