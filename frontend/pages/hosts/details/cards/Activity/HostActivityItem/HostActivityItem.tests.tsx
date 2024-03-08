import React from "react";
import { render, screen } from "@testing-library/react";

import { daysAgo } from "test/test-utils";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";
import createMockActivity from "__mocks__/activityMock";

import HostActivityItem from ".";

describe("HostActivityItem component", () => {
  test("renders the common activity item information (e.g. created at value/tooltip and avatar", () => {
    const mockActivity = createMockActivity({
      created_at: daysAgo(2),
    });

    render(
      <HostActivityItem activity={mockActivity}>
        <></>
      </HostActivityItem>
    );

    expect(screen.getByText("2 days ago")).toBeInTheDocument();
    expect(screen.getByAltText("User avatar")).toBeInTheDocument();
  });

  test("render with default avater when there is no actvity actor email", () => {
    const mockActivity = createMockActivity({
      actor_email: undefined,
    });

    render(
      <HostActivityItem activity={mockActivity}>
        <></>
      </HostActivityItem>
    );

    expect(screen.getByAltText("User avatar")).toHaveAttribute(
      "src",
      DEFAULT_GRAVATAR_LINK
    );
  });

  test("render the users custom avater when there is an actor email", () => {
    const mockActivity = createMockActivity({
      actor_email: "test@email.com",
    });

    render(
      <HostActivityItem activity={mockActivity}>
        <></>
      </HostActivityItem>
    );

    const avatarImageSrc = screen
      .getByAltText("User avatar")
      .getAttribute("src");

    expect(avatarImageSrc).toContain("https://www.gravatar.com");
    expect(avatarImageSrc).not.toEqual(DEFAULT_GRAVATAR_LINK);
  });
});
