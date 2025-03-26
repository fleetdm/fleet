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
        id: `digicert-${cert.name}`,
        name: cert.name,
        description: "DigiCert",
      });
    });
  }

  if (customProxies?.length) {
    customProxies.forEach((cert) => {
      listData.push({
        id: `custom-scep-proxy-${cert.name}`,
        name: cert.name,
        description: "Custom Simple Certificate Enrollment Protocol (SCEP)",
      });
    });
  }

  return listData.sort((a, b) =>
    a.name.toLowerCase().localeCompare(b.name.toLocaleLowerCase())
  );
};

export interface ICertIntegrationNDESWithListId
  extends ICertificatesIntegrationNDES {
  listId: string;
}

export interface ICertIntegrationDigicertWithListId
  extends ICertificatesIntegrationDigicert {
  listId: string;
}

export interface ICertIntegrationCustomSCEPWithListId
  extends ICertificatesIntegrationCustomSCEP {
  listId: string;
}

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
