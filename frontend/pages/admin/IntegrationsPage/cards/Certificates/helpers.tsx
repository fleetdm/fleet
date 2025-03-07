import {
  ICustomScepProxy,
  IDigicertCertificate,
  IScepIntegration,
} from "interfaces/integration";

export interface ICertAuthority {
  name: string;
  certificateAuthority: string;
}

export const generateListData = (
  ncspProxy?: IScepIntegration | null,
  digicertCerts?: IDigicertCertificate[],
  customProxies?: ICustomScepProxy[]
) => {
  const certs: ICertAuthority[] = [];

  // these values for the certificateAuthority is meant to be a hard coded .
  if (ncspProxy) {
    certs.push({
      name: "NDES",
      certificateAuthority:
        "Microsoft Network Device Enrollment Service (NDES)",
    });
  }

  if (digicertCerts?.length) {
    digicertCerts.forEach((cert) => {
      certs.push({
        name: cert.name,
        certificateAuthority: "DigiCert",
      });
    });
  }

  if (customProxies?.length) {
    customProxies.forEach((cert) => {
      certs.push({
        name: cert.name,
        certificateAuthority:
          "Custom Simple Certificate Enrollment Protocol (SCEP)",
      });
    });
  }

  return certs.sort((a, b) =>
    a.name.toLowerCase().localeCompare(b.name.toLocaleLowerCase())
  );
};
