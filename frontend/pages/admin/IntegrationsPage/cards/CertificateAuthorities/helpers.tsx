import {
  ICertificatesIntegrationCustomSCEP,
  ICertificatesIntegrationDigicert,
  ICertificatesIntegrationNDES,
} from "interfaces/integration";

export interface ICertAuthority {
  id: string;
  name: string;
  description: string;
}

export const generateListData = (
  ndesProxy?: ICertificatesIntegrationNDES | null,
  digicertCerts?: ICertificatesIntegrationDigicert[],
  customProxies?: ICertificatesIntegrationCustomSCEP[]
) => {
  const certs: ICertAuthority[] = [];

  // these values for the certificateAuthority is meant to be a hard coded .
  if (ndesProxy) {
    certs.push({
      id: "ndes", // only ever one  NDES so no need to make the id dynamic
      name: "NDES",
      description: "Microsoft Network Device Enrollment Service (NDES)",
    });
  }

  if (digicertCerts?.length) {
    digicertCerts.forEach((cert) => {
      certs.push({
        id: `digicert-${cert.id}`,
        name: cert.name,
        description: "DigiCert",
      });
    });
  }

  if (customProxies?.length) {
    customProxies.forEach((cert) => {
      certs.push({
        id: `custom-scep-proxy-${cert.id}`,
        name: cert.name,
        description: "Custom Simple Certificate Enrollment Protocol (SCEP)",
      });
    });
  }

  return certs.sort((a, b) =>
    a.name.toLowerCase().localeCompare(b.name.toLocaleLowerCase())
  );
};

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
    );
  }

  if (id.includes("custom-scep-proxy")) {
    return customProxies?.find(
      (cert) => id.split("custom-scep-proxy-")[1] === cert.id.toString()
    );
  }

  return null;
};
