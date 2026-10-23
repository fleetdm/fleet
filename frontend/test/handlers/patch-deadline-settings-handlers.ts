import { http, HttpResponse } from "msw";

import {
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import { IPolicy, IPolicySoftwareToInstall } from "interfaces/policy";
import { baseUrl } from "test/test-utils";

const settingsUrl = baseUrl("/software/patch_deadline_settings");
// Inferred wiring from Carlo's Slack response: the Apps subsection pulls from
// the existing patch-automation policies endpoint, not a dedicated endpoint.
// Not spec'd anywhere (see PatchDeadlineSection.tsx for context).
// Note the path segment is /fleets/ not /teams/ even though the endpoints
// constant is still called TEAMS — Fleet renamed the URL but kept the var.
const patchPoliciesUrl = baseUrl("/fleets/:teamId/policies");

export const getPatchDeadlineSettingsHandler = (
  overrides?: Partial<IPatchDeadlineSettings>
) =>
  http.get(settingsUrl, ({ request }) => {
    const url = new URL(request.url);
    const fleetId = Number(url.searchParams.get("fleet_id") ?? 0);
    const body: IPatchDeadlineSettings = {
      fleet_id: fleetId,
      days: ["tuesday"] as Weekday[],
      time: "17:00",
      ...overrides,
    };
    return HttpResponse.json(body);
  });

export const updatePatchDeadlineSettingsHandler = http.patch(
  settingsUrl,
  async ({ request }) => {
    const body = (await request.json()) as IPatchDeadlineSettings;
    return HttpResponse.json(body);
  }
);

export const updatePatchDeadlineSettingsBadRequestHandler = http.patch(
  settingsUrl,
  () =>
    HttpResponse.json(
      {
        message: "Validation error",
        errors: [{ name: "time", reason: "bad" }],
      },
      { status: 400 }
    )
);

// Factory for simulating field-scoped 400 validation errors from the backend.
// Pass per-field reasons to drive the inline-error assertion.
export const updatePatchDeadlineSettingsFieldErrorHandler = (
  fieldErrors: { name: "days" | "time"; reason: string }[]
) =>
  http.patch(settingsUrl, () =>
    HttpResponse.json(
      { message: "Validation error", errors: fieldErrors },
      { status: 400 }
    )
  );

// Hangs the response so tests can assert the Save button's in-flight state.
// Returning a never-resolving promise keeps the mutation stuck in `isLoading`.
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

// Mocks the policies endpoint (?automation_type=patch) response the Apps
// subsection consumes. Pass partial install_software payloads; each becomes
// one patch-type policy row with that install_software attached.
export const getPatchDeadlineAppsHandler = (
  apps: Partial<IPolicySoftwareToInstall>[] = []
) =>
  http.get(patchPoliciesUrl, () => {
    const policies = apps.map((a, i) => {
      const install_software: IPolicySoftwareToInstall = {
        software_title_id: a.software_title_id ?? i + 1,
        name: a.name ?? `App ${i + 1}`,
        ...(a.display_name !== undefined
          ? { display_name: a.display_name }
          : {}),
        ...(a.icon_url !== undefined ? { icon_url: a.icon_url } : {}),
      };
      // Minimal IPolicy shape — only the fields the Settings page reads.
      const policy: Partial<IPolicy> = {
        id: i + 1,
        name: `${install_software.name} patch policy`,
        type: "patch",
        install_software,
      };
      return policy;
    });
    return HttpResponse.json({ policies });
  });

export const emptyPatchDeadlineAppsHandler = getPatchDeadlineAppsHandler([]);

export const getPatchDeadlineAppsErrorHandler = http.get(patchPoliciesUrl, () =>
  HttpResponse.json({ message: "Internal server error" }, { status: 500 })
);
