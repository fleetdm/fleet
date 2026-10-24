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

export type ICertificateAuthorityType =
  | "ndes_scep_proxy"
  | "digicert"
  | "custom_scep_proxy"
  | "hydrant"
  | "smallstep"
  | "custom_est_proxy"
  | "ejbca";

/** all the types of certificates */
export type ICertificateAuthority =
  | ICertificatesNDES
  | ICertificatesDigicert
  | ICertificatesHydrant
  | ICertificatesCustomSCEP
  | ICertificatesSmallstep
  | ICertificatesCustomEST
  | ICertificatesEJBCA;

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

export const isEJBCACertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesEJBCA => {
  return (
    "certificate_authority_name" in integration &&
    "certificate_profile_name" in integration &&
    "end_entity_profile_name" in integration
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

export const isSmallstepCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesSmallstep => {
  return (
    "name" in integration &&
    "url" in integration &&
    "challenge_url" in integration &&
    "username" in integration &&
    "password" in integration
  );
};

export const isCustomESTCertAuthority = (
  integration: ICertificateAuthority
): integration is ICertificatesCustomEST => {
  return (
    "name" in integration &&
    "url" in integration &&
    // differentiates from smallstep
    !("challenge_url" in integration) &&
    "username" in integration &&
    "password" in integration
  );
};

export const createMockCertificateAuthorities = (): ICertificateAuthorityPartial[] => {
  return [
    {
      id: 1,
      name: "Test NDES SCEP Proxy",
      type: "ndes_scep_proxy",
    },
    {
      id: 2,
      name: "Test DigiCert",
      type: "digicert",
    },
    {
      id: 3,
      name: "Test Custom SCEP Proxy",
      type: "custom_scep_proxy",
    },
    {
      id: 4,
      name: "Test Hydrant",
      type: "hydrant",
    },
    {
      id: 5,
      name: "Test Smallstep",
      type: "smallstep",
    },
    {
      id: 6,
      name: "Test Custom EST Proxy",
      type: "custom_est_proxy",
    },
    {
      id: 7,
      name: "Test EJBCA",
      type: "ejbca",
      mtls_cert_expires_at: "2099-01-01T00:00:00Z",
    },
  ];
};
