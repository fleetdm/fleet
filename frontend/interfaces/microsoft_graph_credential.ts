/**
 * A Microsoft Entra app registration Fleet uses to read a tenant's Windows Autopilot registry over Microsoft Graph.
 */
export interface IMicrosoftGraphCredential {
  tenant_id: string;
  client_id: string;
  /**
   * Set by the sync when Entra rejects the credential, and cleared on the next successful sync. It is not set when the
   * credential is saved, because a credential is verified before it is stored.
   */
  credential_invalid: boolean;
  /** When this tenant last synced SUCCESSFULLY. */
  last_synced_at: string | null;
  /** The error from the last sync, when it failed. Null on success. */
  last_sync_error: string | null;
}

/**
 * A credential being written. The API is declarative: send the full list, and an empty list clears the credential.
 * Omitting `client_secret`, or sending the masked placeholder, means "keep the stored secret".
 */
export interface IMicrosoftGraphCredentialFormData
  extends Pick<IMicrosoftGraphCredential, "tenant_id" | "client_id"> {
  client_secret?: string;
}
