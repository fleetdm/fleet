import React from "react";
import { screen, render } from "@testing-library/react";
import { noop } from "lodash";

import { createMockHostEndUser } from "__mocks__/hostMock";

import User from ".";

describe("User card", () => {
  describe("IdP data", () => {
    it("renders the username, full name, groups, and department fields", () => {
      const endUsers = [createMockHostEndUser()];
      render(
        <User
          endUsers={endUsers}
          enableAddEndUser={false}
          onAddEndUser={noop}
        />
      );

      expect(screen.getByText("Username (IdP)")).toBeInTheDocument();
      expect(screen.getByText("jdoe")).toBeInTheDocument();

      expect(screen.getByText("Full name (IdP)")).toBeInTheDocument();
      expect(screen.getByText("John Doe")).toBeInTheDocument();

      expect(screen.getByText("Groups (IdP)")).toBeInTheDocument();
      expect(screen.getByText("GroupA")).toBeInTheDocument();
      expect(screen.getByText("+ 1 more")).toBeInTheDocument();

      expect(screen.getByText("Department (IdP)")).toBeInTheDocument();
      expect(screen.getByText("Engineering")).toBeInTheDocument();
    });
  });

  it("renders the chrome profiles field when has chrome profile values", () => {
    const endUsers = [
      createMockHostEndUser({
        other_emails: [
          { email: "Profile1", source: "google_chrome_profiles" },
          { email: "Profile2", source: "google_chrome_profiles" },
        ],
      }),
    ];
    render(
      <User endUsers={endUsers} enableAddEndUser={false} onAddEndUser={noop} />
    );

    expect(screen.getByText("Google Chrome profiles")).toBeInTheDocument();
    expect(screen.getByText("Profile1")).toBeInTheDocument();
    expect(screen.getAllByText("+ 1 more")).toHaveLength(2); // one for groups, one for Chrome profiles
  });

  it("renders other emails field when has other email values", () => {
    const endUsers = [
      createMockHostEndUser({
        other_emails: [
          { email: "other1@example.com", source: "custom" },
          { email: "other2@example.com", source: "custom" },
        ],
      }),
    ];
    render(
      <User endUsers={endUsers} enableAddEndUser={false} onAddEndUser={noop} />
    );

    expect(screen.getByText("Other emails")).toBeInTheDocument();
    expect(screen.getByText("other1@example.com")).toBeInTheDocument();
    expect(screen.getAllByText("+ 1 more")).toHaveLength(2); // one for groups, one for other emails
  });
});
