import React from "react";
import { render, screen } from "@testing-library/react";
import { createMockHostPastActivity } from "__mocks__/activityMock";

import ReadHostDiskEncryptionKeyActivityItem from "./ReadHostDiskEncryptionKey";

describe("ReadHostDiskEncryptionKeyActivityItem", () => {
  it("renders the activity content", () => {
    render(
      <ReadHostDiskEncryptionKeyActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.getByText("Test User")).toBeVisible();
    expect(screen.getByText(/viewed the disk encryption key/i)).toBeVisible();
  });

  it("does not render the cancel icon", () => {
    render(
      <ReadHostDiskEncryptionKeyActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("does not render the show details icon", () => {
    render(
      <ReadHostDiskEncryptionKeyActivityItem
        activity={createMockHostPastActivity({ actor_full_name: "Test User" })}
        tab="past"
      />
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });
});
