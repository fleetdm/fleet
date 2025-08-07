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

/** This interface represent the smaller subset of cert authority data that is
returned for some of the cert authority endpoints */
export interface ICertificateAuthorityPartial {
  id: number;
  name: string;
  type: ICertificateAuthorityType;
}

export interface ICertificatesNDES {
  id?: number;
  type?: "ndes_scep_proxy";
  url: string;
  admin_url: string;
  username: string;
  password: string;
}

export interface ICertificatesDigicert {
  id?: number;
  type?: "digicert";
  name: string;
  url: string;
  api_token: string;
  profile_id: string;
  certificate_common_name: string;
  certificate_user_principal_names: string[] | null;
  certificate_seat_id: string;
}

export interface ICertificatesHydrant {
  id?: number;
  type?: "hydrant";
  name: string;
  url: string;
  client_id: string;
  client_secret: string;
}

export interface ICertificatesCustomSCEP {
  id?: number;
  type?: "custom_scep_proxy";
  name: string;
  url: string;
  challenge: string;
}

export type ICertificateAuthorityType =
  | "ndes_scep_proxy"
  | "digicert"
  | "custom_scep_proxy"
  | "hydrant";

/** all the types of certificates */
export type ICertificateAuthority =
  | ICertificatesNDES
  | ICertificatesDigicert
  | ICertificatesHydrant
  | ICertificatesCustomSCEP;

export const isNDESCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesNDES => {
  return (
    "admin_url" in integration &&
    "username" in integration &&
    "password" in integration
  );
};

export const isDigicertCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesDigicert => {
  return (
    "profile_id" in integration &&
    "certificate_common_name" in integration &&
    "certificate_user_principal_names" in integration &&
    "certificate_seat_id" in integration
  );
};

export const isHydrantCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesHydrant => {
  return (
    "name" in integration &&
    "url" in integration &&
    "client_id" in integration &&
    "client_secret" in integration
  );
};

export const isCustomSCEPCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesCustomSCEP => {
  return (
    "name" in integration && "url" in integration && "challenge" in integration
  );
};
