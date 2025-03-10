import {
  ICertificatesIntegrationCustomSCEP,
  ICertificatesIntegrationDigicert,
  ICertificatesIntegrationNDES,
} from "interfaces/integration";

export interface ICertAuthorityListData {
  id: string;
  name: string;
  description: string;
}

export const generateListData = (
  ndesProxy?: ICertificatesIntegrationNDES | null,
  digicertCerts?: ICertificatesIntegrationDigicert[],
  customProxies?: ICertificatesIntegrationCustomSCEP[]
) => {
  const listData: ICertAuthorityListData[] = [];

  // these values for the certificateAuthority is meant to be a hard coded .
  if (ndesProxy) {
    listData.push({
      id: "ndes", // only ever one  NDES so no need to make the id dynamic
      name: "NDES",
      description: "Microsoft Network Device Enrollment Service (NDES)",
    });
  }

  if (digicertCerts?.length) {
    digicertCerts.forEach((cert) => {
      listData.push({
        id: `digicert-${cert.id}`,
        name: cert.name,
        description: "DigiCert",
      });
    });
  }

  if (customProxies?.length) {
    customProxies.forEach((cert) => {
      listData.push({
        id: `custom-scep-proxy-${cert.id}`,
        name: cert.name,
        description: "Custom Simple Certificate Enrollment Protocol (SCEP)",
      });
    });
  }

  return listData.sort((a, b) =>
    a.name.toLowerCase().localeCompare(b.name.toLocaleLowerCase())
  );
};

/**
 * Gets the original certificate aithority integration object from the id
 * of the data representing a certificate authority list item.
 */
export const getCertificateAuthority = (
  id: string,
  ncspProxy?: ICertificatesIntegrationNDES | null,
  digicertCerts?: ICertificatesIntegrationDigicert[],
  customProxies?: ICertificatesIntegrationCustomSCEP[]
) => {
  if (id === "ndes") {
    return ncspProxy as ICertificatesIntegrationNDES;
  }

  if (id.includes("digicert")) {
    return digicertCerts?.find(
      (cert) => id.split("digicert-")[1] === cert.id.toString()
    ) as ICertificatesIntegrationDigicert;
  }

  if (id.includes("custom-scep-proxy")) {
    return customProxies?.find(
      (cert) => id.split("custom-scep-proxy-")[1] === cert.id.toString()
    ) as ICertificatesIntegrationCustomSCEP;
  }

  return null;
};
