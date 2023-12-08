import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import AdminWrapper from "./AdminWrapper";

const urlLocation = {
  pathname: "settings/organization/info",
};

// TODO: figure out how to mock the router properly.
const mockRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

describe("AdminWrapper - component", () => {
  it("renders correct navigation for free global admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();

    expect(screen.queryByText(/teams/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for free global maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();

    expect(screen.queryByText(/teams/i)).not.toBeInTheDocument();
  });
  it("renders correct navigation for premium global admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();
    expect(screen.getByText(/teams/i)).toBeInTheDocument();
  });
  it("renders correct navigation for premium global maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();
    expect(screen.getByText(/teams/i)).toBeInTheDocument();
  });
  it("renders correct navigation for premium team admin", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();
    expect(screen.getByText(/teams/i)).toBeInTheDocument();
  });
  it("renders correct navigation for premium team maintainer", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: false,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/organization settings/i)).toBeInTheDocument();
    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/users/i)).toBeInTheDocument();
    expect(screen.getByText(/teams/i)).toBeInTheDocument();
  });
  it("renders correct navigation for sandbox mode", async () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isSandboxMode: true,
        },
      },
    });

    render(
      <AdminWrapper location={urlLocation} router={mockRouter}>
        <></>
      </AdminWrapper>
    );

    expect(screen.getByText(/integrations/i)).toBeInTheDocument();
    expect(screen.getByText(/teams/i)).toBeInTheDocument();

    expect(
      screen.queryByText(/organization settings/i)
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/users/i)).not.toBeInTheDocument();
  });
});
