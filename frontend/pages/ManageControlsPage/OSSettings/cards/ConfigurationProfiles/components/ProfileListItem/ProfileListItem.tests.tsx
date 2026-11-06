import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { IMdmProfile } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";

import ProfileListItem from "./ProfileListItem";

// jsdom has no layout, so control whether the name renders truncated.
jest.mock("hooks/useCheckTruncatedElement", () => ({
  useCheckTruncatedElement: jest.fn(),
}));
const mockedUseCheckTruncatedElement = jest.mocked(useCheckTruncatedElement);

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
  beforeEach(() => {
    mockedUseCheckTruncatedElement.mockReturnValue(false);
  });

  it("shows the user-scope indicator for user-scoped profiles", () => {
    renderItem({ ...baseProfile, scope: "User" });
    expect(screen.getByText("User-scoped")).toBeInTheDocument();
  });

  it("does not show the indicator for system-scoped profiles", () => {
    renderItem({ ...baseProfile, scope: "System" });
    expect(screen.queryByText("User-scoped")).not.toBeInTheDocument();
  });

  it("does not show the indicator for iOS/iPadOS profiles (no user channel)", () => {
    renderItem({ ...baseProfile, platform: "ios", scope: "User" });
    expect(screen.queryByText("User-scoped")).not.toBeInTheDocument();
  });

  it("opens the details modal when the row is clicked", async () => {
    const onClickInfo = jest.fn();
    const { user } = render(
      <ProfileListItem
        isPremium={false}
        profile={baseProfile}
        onClickInfo={onClickInfo}
        onClickEdit={noop}
        onClickDelete={noop}
      />
    );

    await user.click(screen.getByText("Uploaded", { exact: false }));
    expect(onClickInfo).toHaveBeenCalledWith(baseProfile);
  });

  it("does not open the details modal when an action is clicked", async () => {
    const onClickInfo = jest.fn();
    const onClickEdit = jest.fn();
    const { user } = render(
      <ProfileListItem
        isPremium={false}
        profile={baseProfile}
        onClickInfo={onClickInfo}
        onClickEdit={onClickEdit}
        onClickDelete={noop}
      />
    );

    await user.click(screen.getByRole("button", { name: /Edit/ }));
    expect(onClickEdit).toHaveBeenCalled();
    expect(onClickInfo).not.toHaveBeenCalled();
  });

  it("shows no name tooltip when the name fits", async () => {
    const { user } = renderItem(baseProfile);

    await user.hover(screen.getByText("My DDM profile"));

    expect(screen.queryByText(/^Name:/)).not.toBeInTheDocument();
  });

  it("shows the full name on hover when the name is truncated", async () => {
    mockedUseCheckTruncatedElement.mockReturnValue(true);
    const { user } = renderItem(baseProfile);

    await user.hover(screen.getByText("My DDM profile"));

    await waitFor(() => {
      expect(screen.getByText(/^Name:/)).toBeInTheDocument();
    });
    expect(screen.getAllByText("My DDM profile")).toHaveLength(2);
  });
});
