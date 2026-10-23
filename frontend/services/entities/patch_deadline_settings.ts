import { isAxiosError } from "axios";

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

// The GET and PATCH responses both wrap the settings doc in an envelope.
interface IPatchDeadlineSettingsResponse {
  patch_deadline_settings: IPatchDeadlineSettings;
}

// TEMPORARY: when the endpoint 404s (not yet deployed to the dev server), fall
// back to the API defaults so the Settings page is navigable. Delete this
// helper once the backend endpoint is live. Other errors (500s, network
// failures) rethrow so React Query can surface them instead of a silent
// "configured as Tuesday 17:00" that a Save would overwrite with defaults.
const defaultSettings = (fleetId: number): IPatchDeadlineSettings => ({
  fleet_id: fleetId,
  days: ["tuesday"],
  time: "17:00",
});

const is404 = (err: unknown): boolean =>
  isAxiosError(err) && err.response?.status === 404;

export default {
  getSettings: async (fleetId: number): Promise<IPatchDeadlineSettings> => {
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    try {
      const res: IPatchDeadlineSettingsResponse = await sendRequest(
        "GET",
        `${endpoints.PATCH_DEADLINE_SETTINGS}?${queryString}`
      );
      return res.patch_deadline_settings;
    } catch (err) {
      if (!is404(err)) throw err;
      // eslint-disable-next-line no-console
      console.warn(
        "patch_deadline_settings GET 404'd; falling back to defaults:",
        err
      );
      return defaultSettings(fleetId);
    }
  },

  updateSettings: async (
    body: IUpdatePatchDeadlineSettingsBody
  ): Promise<IPatchDeadlineSettings> => {
    const res: IPatchDeadlineSettingsResponse = await sendRequest(
      "PATCH",
      endpoints.PATCH_DEADLINE_SETTINGS,
      body
    );
    return res.patch_deadline_settings;
  },
};
