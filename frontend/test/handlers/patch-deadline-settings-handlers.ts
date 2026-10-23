import { http, HttpResponse } from "msw";

import {
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import { IPolicy, IPolicySoftwareToInstall } from "interfaces/policy";
import { baseUrl } from "test/test-utils";

const settingsUrl = baseUrl("/software/patch_deadline_settings");
// Note the path is /fleets/ even though the endpoints constant is still called
// TEAMS. The URL got renamed, the constant did not.
const patchPoliciesUrl = baseUrl("/fleets/:teamId/policies");

export const getPatchDeadlineSettingsHandler = (
  overrides?: Partial<IPatchDeadlineSettings>
) =>
  http.get(settingsUrl, ({ request }) => {
    const url = new URL(request.url);
    const fleetId = Number(url.searchParams.get("fleet_id") ?? 0);
    const settings: IPatchDeadlineSettings = {
      fleet_id: fleetId,
      days: ["tuesday"] as Weekday[],
      time: "17:00",
      ...overrides,
    };
    return HttpResponse.json({ patch_deadline_settings: settings });
  });

export const updatePatchDeadlineSettingsHandler = http.patch(
  settingsUrl,
  async ({ request }) => {
    const body = (await request.json()) as IPatchDeadlineSettings;
    return HttpResponse.json({ patch_deadline_settings: body });
  }
);

// Factory for simulating field-scoped 400 validation errors.
export const updatePatchDeadlineSettingsFieldErrorHandler = (
  fieldErrors: { name: "days" | "time"; reason: string }[]
) =>
  http.patch(settingsUrl, () =>
    HttpResponse.json(
      { message: "Validation error", errors: fieldErrors },
      { status: 400 }
    )
  );

// Never-resolving response so tests can assert the Save button's in-flight state.
export const updatePatchDeadlineSettingsPendingHandler = http.patch(
  settingsUrl,
  () =>
    new Promise<never>(() => {
      /* intentionally never resolves */
    })
);

export const updatePatchDeadlineSettingsServerErrorHandler = http.patch(
  settingsUrl,
  () => HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);

// Simulates the endpoint not being deployed yet. Service's catch should
// return the API defaults (Tuesday at 17:00) only on 404, letting the page
// render while the backend ships.
export const getPatchDeadlineSettingsNotFoundHandler = http.get(
  settingsUrl,
  () => HttpResponse.json({ message: "Not found" }, { status: 404 })
);

// Non-404 GET failure. Service's catch should rethrow and React Query should
// surface the error so a stale default doesn't masquerade as real settings.
export const getPatchDeadlineSettingsServerErrorHandler = http.get(
  settingsUrl,
  () => HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);

// Each entry becomes a patch-type policy with `patch_deadline: true` and the
// given software under `patch_software`, matching the real BE response shape.
export const getPatchDeadlineAppsHandler = (
  apps: Partial<IPolicySoftwareToInstall>[] = []
) =>
  http.get(patchPoliciesUrl, () => {
    const policies = apps.map((a, i) => {
      const patch_software: IPolicySoftwareToInstall = {
        software_title_id: a.software_title_id ?? i + 1,
        name: a.name ?? `App ${i + 1}`,
        ...(a.display_name !== undefined
          ? { display_name: a.display_name }
          : {}),
        ...(a.icon_url !== undefined ? { icon_url: a.icon_url } : {}),
      };
      const policy: Partial<IPolicy> = {
        id: i + 1,
        name: `${patch_software.name} patch policy`,
        type: "patch",
        patch_deadline: true,
        patch_software,
      };
      return policy;
    });
    return HttpResponse.json({ policies });
  });

export const emptyPatchDeadlineAppsHandler = getPatchDeadlineAppsHandler([]);

export const getPatchDeadlineAppsErrorHandler = http.get(patchPoliciesUrl, () =>
  HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);
