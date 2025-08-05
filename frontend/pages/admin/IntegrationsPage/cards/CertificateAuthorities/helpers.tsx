/**
 * Gets the original certificate aithority integration object from the id
 * of the data representing a certificate authority list item.
 */
// eslint-disable-next-line import/prefer-default-export
export const getCertificateAuthority = (
  id: string,
  ncspProxy?: ICertificatesIntegrationNDES | null,
  digicertCerts?: ICertificatesIntegrationDigicert[],
  customProxies?: ICertificatesIntegrationCustomSCEP[]
) => {
  if (id === "ndes" && ncspProxy) {
    return ncspProxy;
  }

  if (id.includes("digicert") && digicertCerts) {
    return (
      digicertCerts.find((cert) => id.split("digicert-")[1] === cert.name) ??
      null
    );
  }

  if (id.includes("custom-scep-proxy") && customProxies) {
    return (
      customProxies?.find(
        (cert) => id.split("custom-scep-proxy-")[1] === cert.name
      ) ?? null
    );
  }

  return null;
};
