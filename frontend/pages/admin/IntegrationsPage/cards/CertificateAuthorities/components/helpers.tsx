import { ICertificateAuthorityType } from "interfaces/certificates";

const CA_LABEL_BY_TYPE: Record<ICertificateAuthorityType, string> = {
  custom_est_proxy: "Custom Enrollment Over Secure Transport (EST)",
  custom_scep_proxy: "Custom Simple Certificate Enrollment Protocol (SCEP)",
  digicert: "DigiCert",
  hydrant: "Hydrant Enrollment Over Secure Transport (EST)",
  ndes_scep_proxy:
    "Okta CA or Microsoft Network Device Enrollment Service (NDES)",
  smallstep: "Smallstep",
};

export default CA_LABEL_BY_TYPE;
