import {
  IPatchDeadlineApp,
  IPatchDeadlineAppsResponse,
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
// yet, so the Settings page is navigable in dev. Delete these fallbacks once
// the backend endpoints are deployed.
const defaultSettings = (fleetId: number): IPatchDeadlineSettings => ({
  fleet_id: fleetId,
  days: ["tuesday"],
  time: "17:00",
});

// Visible stand-in apps so the subsection renders in dev before the backend
// endpoint is live. 12 entries (so pagination at 10/page is visible), with
// real software_title ids + canonical names from a seeded dev db so each row
// links to a real title details page. Replace with [] (or delete this helper
// outright) alongside the fallback below once the real endpoint ships.
const mockApps = (): IPatchDeadlineAppsResponse => {
  const titles: IPatchDeadlineApp[] = [
    {
      id: 1425,
      name: "1Password",
      bundle_identifier: "com.1password.1password",
    },
    { id: 1336, name: "Discord", bundle_identifier: "com.hnc.Discord" },
    { id: 866, name: "Docker Desktop", bundle_identifier: "com.docker.docker" },
    { id: 272, name: "Figma", bundle_identifier: "com.figma.Desktop" },
    { id: 534, name: "Firefox.app", bundle_identifier: "org.mozilla.firefox" },
    { id: 461, name: "Google Chrome", bundle_identifier: "com.google.Chrome" },
    {
      id: 113,
      name: "Microsoft Edge",
      bundle_identifier: "com.microsoft.edgemac",
    },
    {
      id: 1028,
      name: "Microsoft Teams",
      bundle_identifier: "com.microsoft.teams",
    },
    {
      id: 1310,
      name: "Microsoft Visual Studio Code",
      bundle_identifier: "com.microsoft.VSCode",
    },
    { id: 738, name: "Notion", bundle_identifier: "notion.id" },
    { id: 736, name: "Postman", bundle_identifier: "com.postmanlabs.mac" },
    { id: 835, name: "Slack", bundle_identifier: "com.tinyspeck.slackmacgap" },
  ];
  return {
    apps: titles.map((t) => ({ ...t, source: "apps" })),
    count: titles.length,
  };
};

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

  getApps: async (fleetId: number): Promise<IPatchDeadlineAppsResponse> => {
    const queryString = buildQueryStringFromParams({ fleet_id: fleetId });
    try {
      return await sendRequest(
        "GET",
        `${endpoints.PATCH_DEADLINE_APPS}?${queryString}`
      );
    } catch (err) {
      // TEMPORARY fallback. Remove with the mockApps helper above.
      // eslint-disable-next-line no-console
      console.warn("patch_deadline_settings/apps GET fell back to mocks:", err);
      return mockApps();
    }
  },
};
