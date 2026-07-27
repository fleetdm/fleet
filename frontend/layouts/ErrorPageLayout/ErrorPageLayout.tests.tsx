import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import ErrorPageLayout from "./ErrorPageLayout";

// Both navs pull in a lot of app context/routing internals, so stub them out to
// isolate the layout's authed-vs-unauthed branching.
jest.mock("components/top_nav/SiteTopNav", () => ({
  __esModule: true,
  default: () => <div>site top nav</div>,
}));
jest.mock("components/top_nav/LogoOnlyNav", () => ({
  __esModule: true,
  default: () => <div>logo only nav</div>,
}));

describe("ErrorPageLayout", () => {
  const router = createMockRouter();
  const location = { pathname: "/404", search: "", query: {} };

  it("renders the logo-only nav when there is no authenticated user", () => {
    const render = createCustomRenderer();

    render(
      <ErrorPageLayout router={router} location={location}>
        <p>error content</p>
      </ErrorPageLayout>
    );

    expect(screen.getByText("logo only nav")).toBeInTheDocument();
    expect(screen.queryByText("site top nav")).not.toBeInTheDocument();
    expect(screen.getByText("error content")).toBeInTheDocument();
  });

  it("renders the full top nav when a user is authenticated", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          currentUser: createMockUser(),
          config: createMockConfig(),
        },
      },
    });

    render(
      <ErrorPageLayout router={router} location={location}>
        <p>error content</p>
      </ErrorPageLayout>
    );

    expect(screen.getByText("site top nav")).toBeInTheDocument();
    expect(screen.queryByText("logo only nav")).not.toBeInTheDocument();
  });
});
