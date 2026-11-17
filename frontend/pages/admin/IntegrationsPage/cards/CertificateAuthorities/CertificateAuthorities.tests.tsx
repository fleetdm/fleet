import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import CertificateAuthorities from "./CertificateAuthorities";

const caListHandler = (certAuthorities: ICertificateAuthorityPartial[]) =>
  http.get(baseUrl("/certificate_authorities"), () =>
    HttpResponse.json({ certificate_authorities: certAuthorities })
  );

const renderPage = () =>
  createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier: true } },
  })(<CertificateAuthorities />);

describe("CertificateAuthorities", () => {
  it("warns about CAs that share a name from before names had to be unique", async () => {
    mockServer.use(
      caListHandler([
        { id: 1, name: "WIFI", type: "digicert" },
        { id: 2, name: "wifi", type: "custom_scep_proxy" },
        { id: 3, name: "VPN", type: "hydrant" },
      ])
    );
    renderPage();

    expect(
      await screen.findByText(
        'These certificate authorities have conflicting names: "digicert:WIFI" "custom_scep_proxy:wifi". Please rename one of them and update any profiles that use it.'
      )
    ).toBeInTheDocument();
  });

  it("doesn't warn when the names are unique", async () => {
    mockServer.use(
      caListHandler([
        { id: 1, name: "WIFI", type: "digicert" },
        { id: 3, name: "VPN", type: "hydrant" },
      ])
    );
    renderPage();

    expect(await screen.findByText("WIFI")).toBeInTheDocument();
    expect(
      screen.queryByText(/have conflicting names/)
    ).not.toBeInTheDocument();
  });
});
