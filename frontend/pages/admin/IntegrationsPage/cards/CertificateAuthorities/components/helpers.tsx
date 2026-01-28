import { ICertificateAuthorityType } from "interfaces/certificates";

const CA_LABEL_BY_TYPE: Record<ICertificateAuthorityType, string> = {
  custom_est_proxy: "Custom EST (Enrollment Over Secure Transport)",
  custom_scep_proxy: "Custom SCEP (Simple Certificate Enrollment Protocol)",
  digicert: "DigiCert",
  hydrant: "Hydrant EST (Enrollment Over Secure Transport)",
  ndes_scep_proxy:
    "Okta CA or Microsoft NDES (Network Device Enrollment Service)",
  smallstep: "Smallstep",
};

export default CA_LABEL_BY_TYPE;
