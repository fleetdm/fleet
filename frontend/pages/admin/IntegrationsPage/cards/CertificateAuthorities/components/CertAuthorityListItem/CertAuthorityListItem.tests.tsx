import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { ICertAuthorityListData } from "../CertificateAuthorityList/CertificateAuthorityList";

import CertAuthorityListItem from "./CertAuthorityListItem";

const createListData = (
  overrides?: Partial<ICertAuthorityListData>
): ICertAuthorityListData => ({
  id: 1,
  name: "TEST_CA",
  type: "digicert",
  description: "DigiCert",
  ...overrides,
});

describe("CertAuthorityListItem", () => {
  it("shows when the client certificate of an EJBCA CA expires", () => {
    render(
      <CertAuthorityListItem
        cert={createListData({
          type: "ejbca",
          description: "EJBCA",
          mtls_cert_expires_at: "2099-01-01T00:00:00Z",
        })}
        onClickEdit={noop}
        onClickDelete={noop}
      />
    );

    expect(screen.getByText(/Client certificate expires/)).toBeVisible();
    expect(screen.getByText("January 1, 2099")).toBeVisible();
  });

  it("shows no expiry for a CA that has no client certificate", () => {
    render(
      <CertAuthorityListItem
        cert={createListData()}
        onClickEdit={noop}
        onClickDelete={noop}
      />
    );

    expect(screen.queryByText(/Client certificate expires/)).toBeNull();
  });
});
