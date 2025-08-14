import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { ISecret } from "interfaces/secrets";
import { UserEvent } from "@testing-library/user-event";
import { IScript } from "interfaces/script";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import Secrets from "./Secrets";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

describe("Custom variables", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
  });

  describe("empty state", () => {
    it("renders when no secrets are saved", () => {});
  });
  describe("non-empty state", () => {
    const mockSecrets: ISecret[] = [
      {
        name: "Secret uno",
        id: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      {
        name: "Secret dos",
        id: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];
    const secretsResponse: { secrets: ISecret[] } = { secrets: [] };
    // Mock the scripts endpoint to return our two test scripts.
    const secretsHandler = http.get(baseUrl("/custom_variables"), () => {
      return HttpResponse.json({
        custom_variables: secretsResponse.secrets,
        count: mockSecrets.length,
        has_prev_results: false,
        has_next_results: false,
      });
    });
    beforeEach(() => {
      mockServer.use(secretsHandler);
      secretsResponse.secrets = [...mockSecrets];
    });

    it("renders when secrets are saved", async () => {
      render(<Secrets />);
      await waitFor(() => {
        expect(screen.getByText("SECRET UNO")).toBeInTheDocument();
        expect(screen.getByText("SECRET DOS")).toBeInTheDocument();
      });
    });
  });
});
