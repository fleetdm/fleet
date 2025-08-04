import { IListSort } from "./list_options";

export interface IHostCertificate {
  id: number;
  not_valid_after: string;
  not_valid_before: string;
  certificate_authority: boolean;
  common_name: string;
  key_algorithm: string;
  key_strength: number;
  key_usage: string;
  serial: string;
  signing_algorithm: string;
  subject: {
    country: string;
    organization: string;
    organizational_unit: string;
    common_name: string;
  };
  issuer: {
    country: string;
    organization: string;
    organizational_unit: string;
    common_name: string;
  };
  source: string;
  username: string;
}

export const CERTIFICATES_DEFAULT_SORT: IListSort = {
  order_key: "common_name",
  order_direction: "asc",
} as const;

export interface ICertificatesIntegrationNDES {
  url: string;
  admin_url: string;
  username: string;
  password: string;
}

export interface ICertificatesIntegrationDigicert {
  name: string;
  url: string;
  api_token: string;
  profile_id: string;
  certificate_common_name: string;
  certificate_user_principal_names: string[] | null;
  certificate_seat_id: string;
}

export interface ICertificatesIntegrationHydrant {
  name: string;
  url: string;
  client_id: string;
  client_secret: string;
}

export interface ICertificatesIntegrationCustomSCEP {
  name: string;
  url: string;
  challenge: string;
}

export type ICertificateAuthorityType =
  | "ndes"
  | "digicert"
  | "custom"
  | "hydrant";

/** all the types of certificate integrations */
export type ICertificateIntegration =
  | ICertificatesIntegrationNDES
  | ICertificatesIntegrationDigicert
  | ICertificatesIntegrationHydrant
  | ICertificatesIntegrationCustomSCEP;

export const isNDESCertIntegration = (
  integration: ICertificateIntegration
): integration is ICertificatesIntegrationNDES => {
  return (
    "admin_url" in integration &&
    "username" in integration &&
    "password" in integration
  );
};

export const isDigicertCertIntegration = (
  integration: ICertificateIntegration
): integration is ICertificatesIntegrationDigicert => {
  return (
    "profile_id" in integration &&
    "certificate_common_name" in integration &&
    "certificate_user_principal_names" in integration &&
    "certificate_seat_id" in integration
  );
};

export const isHydrantCertIntegration = (
  integration: ICertificateIntegration
): integration is ICertificatesIntegrationHydrant => {
  return (
    "name" in integration &&
    "url" in integration &&
    "client_id" in integration &&
    "client_secret" in integration
  );
};

export const isCustomSCEPCertIntegration = (
  integration: ICertificateIntegration
): integration is ICertificatesIntegrationCustomSCEP => {
  return (
    "name" in integration && "url" in integration && "challenge" in integration
  );
};
