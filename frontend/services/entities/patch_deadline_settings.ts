import {
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IUpdatePatchDeadlineSettingsBody {
  fleet_id: number;
  days: Weekday[];
  time: string;
}

// TEMPORARY: fall back to the API defaults when the real endpoint isn't live
// yet, so the Settings page is navigable in dev. Delete this helper once the
// backend endpoints are deployed.
const defaultSettings = (fleetId: number): IPatchDeadlineSettings => ({
  fleet_id: fleetId,
  days: ["tuesday"],
  time: "17:00",
});

export default {
  getSettings: async (fleetId: number): Promise<IPatchDeadlineSettings> => {
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    try {
      return await sendRequest(
        "GET",
        `${endpoints.PATCH_DEADLINE_SETTINGS}?${queryString}`
      );
    } catch (err) {
      // TEMPORARY fallback. Remove with the defaultSettings helper above.
      // eslint-disable-next-line no-console
      console.warn("patch_deadline_settings GET fell back to defaults:", err);
      return defaultSettings(fleetId);
    }
  },

  updateSettings: (
    body: IUpdatePatchDeadlineSettingsBody
  ): Promise<IPatchDeadlineSettings> => {
    return sendRequest("PATCH", endpoints.PATCH_DEADLINE_SETTINGS, body);
  },
};
