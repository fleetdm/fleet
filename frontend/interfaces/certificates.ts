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
  /** The expiry of the client certificate of an EJBCA CA. Only EJBCA CAs have it. */
  mtls_cert_expires_at?: string;
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

export interface ICertificatesSmallstep {
  id?: number;
  type?: "smallstep";
  name: string;
  url: string;
  challenge_url: string;
  username: string;
  password: string;
}

export interface ICertificatesCustomEST {
  id?: number;
  type?: "custom_est_proxy";
  name: string;
  url: string;
  username: string;
  password: string;
}

export interface ICertificatesEJBCA {
  id?: number;
  type?: "ejbca";
  name: string;
  url: string;
  /** The client certificate and key as a base64 PKCS#12. It is only sent, never returned. */
  client_p12?: string;
  /** The password of `client_p12`. It is only sent, never returned. */
  password?: string;
  trust_ca_bundle?: string;
  certificate_authority_name: string;
  certificate_profile_name: string;
  end_entity_profile_name: string;
  certificate_common_name: string;
  username_template?: string;
  certificate_user_principal_names: string[] | null;
  /** When the stored client certificate expires. It is returned, never sent. */
  mtls_cert_expires_at?: string;
}

export interface ICertificatesGoogleCAS {
  id?: number;
  type?: "google_cas";
  name: string;
  project: string;
  location: string;
  ca_pool: string;
  /** The service account key file's JSON. The API returns it masked. */
  api_key_json: string;
}

export type ICertificateAuthorityType =
  | "ndes_scep_proxy"
  | "digicert"
  | "custom_scep_proxy"
  | "hydrant"
  | "smallstep"
  | "custom_est_proxy"
  | "ejbca"
  | "google_cas";

/** all the types of certificates */
export type ICertificateAuthority =
  | ICertificatesNDES
  | ICertificatesDigicert
  | ICertificatesHydrant
  | ICertificatesCustomSCEP
  | ICertificatesSmallstep
  | ICertificatesCustomEST
  | ICertificatesEJBCA
  | ICertificatesGoogleCAS;

export const isEJBCACertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesEJBCA => {
  return (
    "certificate_authority_name" in integration &&
    "certificate_profile_name" in integration &&
    "end_entity_profile_name" in integration
  );
};
