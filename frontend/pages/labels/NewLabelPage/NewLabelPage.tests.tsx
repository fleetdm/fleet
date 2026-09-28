import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import {
  EMPTY_QUERY_ERR,
  validateQuery,
} from "components/forms/validators/validate_query";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import NewLabelPage from "./NewLabelPage";

// Ace can't be typed into under jsdom; a textarea exposes the same
// value/onChange/error contract.
jest.mock("components/SQLEditor", () => ({
  __esModule: true,
  default: ({
    name,
    value,
    error,
    onChange,
  }: {
    name?: string;
    value: string;
    error?: string;
    onChange?: (value: string) => void;
  }) => (
    <>
      {error && <span>{error}</span>}
      <textarea
        data-testid={`sql-editor-${name}`}
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
    </>
  ),
}));

const routerProps = {
  location: { pathname: "/labels/new/dynamic", query: {} },
  params: {},
  route: {},
  router: { push: jest.fn(), replace: jest.fn() },
  routeParams: {},
  routes: [],
} as any; // eslint-disable-line @typescript-eslint/no-explicit-any

const syntaxErrorFor = (sql: string) => validateQuery(sql).error as string;

const renderPage = () => {
  mockServer.use(
    http.get(baseUrl("/custom_host_vitals"), () =>
      HttpResponse.json({ custom_host_vitals: [], count: 0 })
    ),
    http.get(baseUrl("/scim/details"), () =>
      HttpResponse.json({ last_request: null })
    )
  );
  const render = createCustomRenderer({ withBackendMock: true });
  return render(<NewLabelPage {...routerProps} />);
};

describe("NewLabelPage dynamic label query", () => {
  it("shows a syntax error for an invalid query without disabling Save", async () => {
    const { user } = renderPage();

    const editor = screen.getByTestId("sql-editor-query");
    await user.clear(editor);
    await user.type(editor, "SELEC * FRM users");

    expect(
      await screen.findByText(syntaxErrorFor("SELEC * FRM users"))
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("clears the syntax error once the query is fixed", async () => {
    const { user } = renderPage();

    const editor = screen.getByTestId("sql-editor-query");
    await user.clear(editor);
    await user.type(editor, "SELEC 1");
    await screen.findByText(syntaxErrorFor("SELEC 1"));

    await user.clear(editor);
    await user.type(editor, "SELECT 1");
    await waitFor(() => {
      expect(
        screen.queryByText(syntaxErrorFor("SELEC 1"))
      ).not.toBeInTheDocument();
    });
  });

  it("still shows the required-query error, not a syntax error, when cleared", async () => {
    const { user } = renderPage();

    const editor = screen.getByTestId("sql-editor-query");
    await user.clear(editor);
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText(EMPTY_QUERY_ERR)).toBeInTheDocument();
  });
});
