import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import CertificateAuthorities from "./CertificateAuthorities";

describe("CertificateAuthorities", () => {
  it("warns about CAs that share a name from before names had to be unique", async () => {
    mockServer.use(
      http.get(baseUrl("/certificate_authorities"), () =>
        HttpResponse.json({
          certificate_authorities: [
            { id: 1, name: "WIFI", type: "digicert" },
            { id: 2, name: "wifi", type: "custom_scep_proxy" },
            { id: 3, name: "VPN", type: "hydrant" },
          ],
        })
      )
    );
    createCustomRenderer({
      withBackendMock: true,
      context: { app: { isPremiumTier: true } },
    })(<CertificateAuthorities />);

    expect(
      await screen.findByText(
        'These certificate authorities have conflicting names: "digicert:WIFI" "custom_scep_proxy:wifi". Please rename one of them and update any profiles that use it.'
      )
    ).toBeInTheDocument();
  });
});
