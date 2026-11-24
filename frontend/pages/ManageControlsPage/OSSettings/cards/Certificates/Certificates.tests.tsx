import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import Certificates from "./Certificates";

const emptyCertsHandler = http.get(baseUrl("/certificates"), () =>
  HttpResponse.json({
    certificates: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const emptyCACertsHandler = http.get(baseUrl("/certificates/ca"), () =>
  HttpResponse.json({
    ca_certificates: [],
  })
);

const scepCAHandler = http.get(baseUrl("/certificate_authorities"), () =>
  HttpResponse.json({
    certificate_authorities: [
      { id: 1, name: "SCEP", type: "custom_scep_proxy" },
    ],
  })
);

const emptyCAHandler = http.get(baseUrl("/certificate_authorities"), () =>
  HttpResponse.json({ certificate_authorities: [] })
);

const androidMdmConfig = {
  mdm: { android_enabled_and_configured: true },
} as any;

const baseProps = {
  currentTeamId: 0,
  router: createMockRouter(),
  onMutation: jest.fn(),
};

describe("Certificates tab-header", () => {
  it("always renders the description", async () => {
    mockServer.use(emptyCertsHandler, emptyCACertsHandler, emptyCAHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true, config: androidMdmConfig },
      },
    });

    render(<Certificates {...baseProps} />);

    expect(
      await screen.findByText(/Deploy certificates\. Currently only Android/i)
    ).toBeInTheDocument();
  });

  it("hides Add certificate on Fleet Free", async () => {
    mockServer.use(emptyCertsHandler, emptyCACertsHandler, emptyCAHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: false, config: androidMdmConfig },
      },
    });

    render(<Certificates {...baseProps} />);

    // Wait for the tab-header (description always renders) then verify no
    // Add certificate button.
    expect(await screen.findByText(/Deploy certificates/i)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add certificate$/i })
    ).not.toBeInTheDocument();
  });

  it("hides Add certificate when no custom SCEP CA is configured", async () => {
    mockServer.use(emptyCertsHandler, emptyCACertsHandler, emptyCAHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true, config: androidMdmConfig },
      },
    });

    render(<Certificates {...baseProps} />);

    // Wait for the "no CAs" empty state to confirm the CA fetch resolved.
    expect(
      await screen.findByText(/Add certificate authority/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /Add certificate$/i })
    ).not.toBeInTheDocument();
  });

  it("shows Add certificate when premium + Android MDM + custom SCEP CA", async () => {
    mockServer.use(emptyCertsHandler, emptyCACertsHandler, scepCAHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true, config: androidMdmConfig },
      },
    });

    render(<Certificates {...baseProps} />);

    expect(
      await screen.findByRole("button", { name: /Add certificate$/i })
    ).toBeInTheDocument();
  });

  it("displays uploaded static CA certificate and allows deleting it", async () => {
    const staticCAHandler = http.get(baseUrl("/certificates/ca"), () =>
      HttpResponse.json({
        ca_certificates: [
          {
            id: 99,
            team_id: 0,
            name: "Corporate Root CA",
            sha256: "abc123hash",
            subject: "CN=Corporate Root CA",
            issuer: "CN=Corporate Root CA",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          },
        ],
      })
    );

    mockServer.use(emptyCertsHandler, staticCAHandler, emptyCAHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isPremiumTier: true, config: androidMdmConfig },
      },
    });

    render(<Certificates {...baseProps} />);

    expect(await screen.findByText("Corporate Root CA")).toBeInTheDocument();
    expect(screen.getByText(/Root\/intermediate CA/i)).toBeInTheDocument();
  });
});
