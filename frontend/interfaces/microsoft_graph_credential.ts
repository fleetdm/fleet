/**
 * A Microsoft Entra app registration Fleet uses to read a tenant's Windows Autopilot registry over Microsoft Graph.
 *
 * Fleet stores at most one. The client secret is never returned by the API: `GET` omits the field entirely, so a stored
 * secret is only knowable by the presence of the credential itself.
 */
export interface IMicrosoftGraphCredential {
  tenant_id: string;
  client_id: string;
  /**
   * Set by the sync when Entra rejects the credential, and cleared on the next successful sync. It is not set when the
   * credential is saved, because a credential is verified before it is stored.
   */
  credential_invalid: boolean;
  /** When this tenant last synced SUCCESSFULLY. Null until the first sync succeeds, and not advanced by a failed pass,
   * so a stale value next to `last_sync_error` shows how far behind the pending hosts have fallen. */
  last_synced_at: string | null;
  /** The error from the last sync, when it failed. Null on success. Already classified into one admin-facing sentence
   * server-side, so it renders verbatim. */
  last_sync_error: string | null;
}

/**
 * A credential being written. The API is declarative: send the full list, and an empty list clears the credential.
 *
 * Omitting `client_secret`, or sending the masked placeholder, means "keep the stored secret".
 */
export interface IMicrosoftGraphCredentialFormData {
  tenant_id: string;
  client_id: string;
  client_secret?: string;
}
