import { createMockCertificateAuthorityPartial } from "__mocks__/certificatesMock";

import { getEarliestEJBCACertExpiry } from "./App";

describe("getEarliestEJBCACertExpiry", () => {
  it("is the earliest client certificate expiry of the EJBCA certificate authorities", () => {
    expect(
      getEarliestEJBCACertExpiry([
        createMockCertificateAuthorityPartial({
          type: "ejbca",
          mtls_cert_expires_at: "2030-05-01T00:00:00Z",
        }),
        createMockCertificateAuthorityPartial({
          type: "ejbca",
          mtls_cert_expires_at: "2028-02-01T00:00:00Z",
        }),
        // Other types and EJBCA CAs without an expiry don't count.
        createMockCertificateAuthorityPartial({
          type: "digicert",
          mtls_cert_expires_at: "2020-01-01T00:00:00Z",
        }),
        createMockCertificateAuthorityPartial({ type: "ejbca" }),
      ])
    ).toBe("2028-02-01T00:00:00.000Z");
  });

  it("is empty when there is no EJBCA certificate authority with an expiry", () => {
    expect(
      getEarliestEJBCACertExpiry([
        createMockCertificateAuthorityPartial({ type: "digicert" }),
      ])
    ).toBe("");
    expect(getEarliestEJBCACertExpiry([])).toBe("");
  });
});
