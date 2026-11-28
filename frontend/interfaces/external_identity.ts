/** An identity from an `auth.jwt_issuers` issuer, bound to an API-only user. */
export interface IExternalIdentity {
  /** Name of the configured issuer. */
  issuer: string;
  /** Exact, case-sensitive value of the issuer's subject claim. */
  subject: string;
}

export interface IExternalIdentitiesResponse {
  external_identities: IExternalIdentity[];
  /** Configured issuer names. Only the GET response includes them. */
  issuers?: string[];
}
