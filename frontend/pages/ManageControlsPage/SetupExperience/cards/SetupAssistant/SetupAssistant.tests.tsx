import React from "react";
import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import { createGetConfigHandler } from "test/handlers/config-handlers";
import { createGetTeamHandler } from "test/handlers/team-handlers";
import { createMockMdmConfig } from "__mocks__/configMock";

import SetupAssistant from "./SetupAssistant";

const enrollmentProfileUrl = baseUrl("/enrollment_profiles/automatic");
const defaultEnrollmentProfileUrl = baseUrl(
  "/enrollment_profiles/automatic/default"
);

const setupMdmConfigured = () => {
  mockServer.use(createGetConfigHandler());
  mockServer.use(createGetTeamHandler({}));
};

const setupMdmNotConfigured = () => {
  mockServer.use(
    createGetConfigHandler({
      mdm: createMockMdmConfig({ enabled_and_configured: false }),
    })
  );
  mockServer.use(createGetTeamHandler({}));
};

describe("SetupAssistant", () => {
  it("renders the page description on the empty state when MDM isn't configured", async () => {
    setupMdmNotConfigured();
    mockServer.use(
      http.get(enrollmentProfileUrl, () => {
        return new HttpResponse("Not found", { status: 404 });
      }),
      http.get(defaultEnrollmentProfileUrl, () => {
        return HttpResponse.json({
          enrollment_profile: {},
        });
      })
    );
    mockServer.use(
      http.get(defaultEnrollmentProfileUrl, () => {
        return HttpResponse.json({
          enrollment_profile: { is_mandatory: true },
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<SetupAssistant router={createMockRouter()} currentTeamId={1} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Additional configuration required/)
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText(/Turn on MDM and automatic enrollment to customize/)
    ).toBeVisible();
  });

  it("renders the profile uploader when MDM is configured and no profile is uploaded", async () => {
    setupMdmConfigured();
    mockServer.use(
      http.get(enrollmentProfileUrl, () => {
        return new HttpResponse("Not found", { status: 404 });
      })
    );
    mockServer.use(
      http.get(defaultEnrollmentProfileUrl, () => {
        return HttpResponse.json({
          enrollment_profile: { is_mandatory: true },
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<SetupAssistant router={createMockRouter()} currentTeamId={1} />);

    expect(
      await screen.findByRole("button", { name: "Add profile" })
    ).toBeVisible();
    expect(
      screen.getByText(/Add an automatic enrollment profile/)
    ).toBeVisible();
  });

  it("renders the profile card when a profile has been uploaded", async () => {
    setupMdmConfigured();
    mockServer.use(
      http.get(enrollmentProfileUrl, () => {
        return HttpResponse.json({
          team_id: 1,
          name: "test-profile.json",
          uploaded_at: "2024-01-01T00:00:00Z",
          enrollment_profile: {},
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
    });

    render(<SetupAssistant router={createMockRouter()} currentTeamId={1} />);

    expect(await screen.findByText("test-profile.json")).toBeVisible();
    expect(
      screen.getByText(/Add an automatic enrollment profile/)
    ).toBeVisible();
  });
});
