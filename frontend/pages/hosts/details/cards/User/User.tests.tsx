import React from "react";
import { screen, render } from "@testing-library/react";
import { noop } from "lodash";

import { createMockHostEndUser } from "__mocks__/hostMock";

import User from ".";

describe("User card", () => {
  it("renders the username field when the platform is Apple", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Username (IdP)")).toBeInTheDocument();
    expect(screen.getByText("jdoe")).toBeInTheDocument();
  });

  it("renders the username field when the platform is android", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="android"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Username (IdP)")).toBeInTheDocument();
    expect(screen.getByText("jdoe")).toBeInTheDocument();
  });

  it("renders the full name field when the platform is Apple and has full name values", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Full name (IdP)")).toBeInTheDocument();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders the full name field when the platform is Android and has full name values", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Full name (IdP)")).toBeInTheDocument();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders the groups field when the platform is Apple and has groups values", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Groups (IdP)")).toBeInTheDocument();
    expect(screen.getByText("GroupA")).toBeInTheDocument();
    expect(screen.getByText("+ 1 more")).toBeInTheDocument();
  });

  it("renders the groups field when the platform is Android and has groups values", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Groups (IdP)")).toBeInTheDocument();
    expect(screen.getByText("GroupA")).toBeInTheDocument();
    expect(screen.getByText("+ 1 more")).toBeInTheDocument();
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
      <User
        platform="windows"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Google Chrome profiles")).toBeInTheDocument();
    expect(screen.getByText("Profile1")).toBeInTheDocument();
    expect(screen.getByText("+ 1 more")).toBeInTheDocument();
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
      <User
        platform="windows"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Other emails")).toBeInTheDocument();
    expect(screen.getByText("other1@example.com")).toBeInTheDocument();
    expect(screen.getByText("+ 1 more")).toBeInTheDocument();
  });

  it("renders the department field when the platform is Apple and it has department value", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="darwin"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Department (IdP)")).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
  });

  it("renders the department field when the platform is Android and it has department value", () => {
    const endUsers = [createMockHostEndUser()];
    render(
      <User
        platform="android"
        endUsers={endUsers}
        enableAddEndUser={false}
        onAddEndUser={noop}
      />
    );

    expect(screen.getByText("Department (IdP)")).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
  });
});
