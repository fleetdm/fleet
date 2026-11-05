import sendRequest, { sendRequestWithProgress } from "services";
import endpoints from "utilities/endpoints";

export type IAutofillPolicy = {
  description: string;
  resolution: string;
};

export type AutofillConfigProfilePlatform = "apple" | "windows";

/** Matches the API's `type`, which is also a ProfileContentType key. */
export type AutofillConfigProfileType =
  | "mobileconfig"
  | "declaration"
  | "windows";

export interface IAutofillConfigProfileNameDescriptionResponse {
  name: string;
  description: string;
}

export interface IAutofillConfigProfileContentResponse {
  type: AutofillConfigProfileType;
  /** The root PayloadDisplayName of a .mobileconfig, otherwise null. */
  name: string | null;
  content: string;
  delivery_notes: string;
}

export interface IAutofillConfigProfileParams {
  instructions: string;
  platform: AutofillConfigProfilePlatform;
  signal?: AbortSignal;
}

export interface IAutofillConfigProfileContentParams
  extends IAutofillConfigProfileParams {
  /** Left out for Unassigned and on Fleet Free. */
  fleetId?: number;
}

export default {
  getPolicyInterpretationFromSQL: (sql: string): Promise<IAutofillPolicy> => {
    const { AUTOFILL_POLICY } = endpoints;

    return sendRequest("POST", AUTOFILL_POLICY, { sql });
  },

  getConfigProfileNameDescription: ({
    instructions,
    platform,
    signal,
  }: IAutofillConfigProfileParams): Promise<IAutofillConfigProfileNameDescriptionResponse> =>
    sendRequestWithProgress({
      method: "POST",
      path: endpoints.AUTOFILL_CONFIG_PROFILE_NAME_DESCRIPTION,
      data: { natural_language_instructions: instructions, platform },
      signal,
    }),

  getConfigProfileContent: ({
    instructions,
    platform,
    fleetId,
    signal,
  }: IAutofillConfigProfileContentParams): Promise<IAutofillConfigProfileContentResponse> =>
    sendRequestWithProgress({
      method: "POST",
      path: endpoints.AUTOFILL_CONFIG_PROFILE_CONTENT,
      data: {
        natural_language_instructions: instructions,
        platform,
        fleet_id: fleetId,
      },
      signal,
    }),
};
