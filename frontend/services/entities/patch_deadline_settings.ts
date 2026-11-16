import {
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IUpdatePatchDeadlineSettingsFormData {
  fleet_id: number;
  days: Weekday[];
  time: string;
}

// The GET and PATCH responses both wrap the settings doc in an envelope.
interface IPatchDeadlineSettingsResponse {
  patch_deadline_settings: IPatchDeadlineSettings;
}

export default {
  getSettings: async (fleetId: number): Promise<IPatchDeadlineSettings> => {
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    const res: IPatchDeadlineSettingsResponse = await sendRequest(
      "GET",
      `${endpoints.PATCH_DEADLINE_SETTINGS}?${queryString}`
    );
    return res.patch_deadline_settings;
  },

  updateSettings: async (
    body: IUpdatePatchDeadlineSettingsFormData
  ): Promise<IPatchDeadlineSettings> => {
    const res: IPatchDeadlineSettingsResponse = await sendRequest(
      "PATCH",
      endpoints.PATCH_DEADLINE_SETTINGS,
      body
    );
    return res.patch_deadline_settings;
  },
};
