import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import {
  createMockSoftwarePackage,
  createMockSoftwareTitleDetails,
} from "__mocks__/softwareMock";
import { ISoftwarePackage } from "interfaces/software";

// Per-title package cap exposed by #48396's backend guard. Mirrored here so
// tests don't have to hard-code the literal in multiple places.
export const MAX_PACKAGES_PER_TITLE = 10;

const titleUrl = baseUrl("/software/titles/:id");
const addPackageUrl = baseUrl("/software/package");
const editPackageUrl = baseUrl("/software/titles/:id/package");
const deletePackageUrl = baseUrl("/software/titles/:id/available_for_install");

/** Build an `n`-package fixture list. Each package gets a distinct
 * `installer_id`, name, version, and hash so assertions on per-installer
 * targeting are unambiguous. */
export const createMockPackages = (
  n: number,
  overrides?: (i: number) => Partial<ISoftwarePackage>
): ISoftwarePackage[] =>
  Array.from({ length: n }, (_, i) =>
    createMockSoftwarePackage({
      installer_id: i + 1,
      title_id: 1,
      name: `TestPackage-1.${i}.0.pkg`,
      version: `1.${i}.0`,
      hash_sha256: `hash${i + 1}`,
      ...overrides?.(i),
    })
  );

/** Build a title-details fixture whose `software_package` is always derived
 * from `packages[0]`. Use this anywhere a test needs both fields to stay in
 * lockstep so mutating `packages` can't silently drift the alias. */
export const buildTitleWithPackages = (
  packages: ISoftwarePackage[],
  titleOverrides?: Parameters<typeof createMockSoftwareTitleDetails>[0]
) =>
  createMockSoftwareTitleDetails({
    software_package: packages[0] ?? null,
    packages,
    ...titleOverrides,
  });

// GET /software/titles/:id — multi-package response.
export const getMultiPackageTitleHandler = (
  packageCount = 2,
  titleOverrides?: Parameters<typeof createMockSoftwareTitleDetails>[0]
) =>
  http.get(titleUrl, ({ params }) => {
    const packages = createMockPackages(packageCount, () => ({
      title_id: Number(params.id),
    }));
    return HttpResponse.json({
      software_title: buildTitleWithPackages(packages, {
        id: Number(params.id),
        ...titleOverrides,
      }),
    });
  });

// GET /software/titles/:id — title with no packages.
export const getEmptyPackagesTitleHandler = http.get(titleUrl, ({ params }) =>
  HttpResponse.json({
    software_title: createMockSoftwareTitleDetails({
      id: Number(params.id),
      software_package: null,
      packages: [],
    }),
  })
);

// POST /software/package — happy path adds to an existing title.
export const addPackageHandler = http.post(addPackageUrl, async ({ request }) =>
  HttpResponse.json({
    software_title_id: Number(
      (await request.formData()).get("software_title_id") ?? 0
    ),
  })
);

// POST /software/package — duplicate-hash rejection. Copy is verbatim from
// Figma page 2:130 / issue #48400.
export const addPackageDuplicateHashHandler = http.post(
  addPackageUrl,
  async ({ request }) => {
    const filename =
      ((await request.formData()).get("software") as File | null)?.name ?? "";
    return HttpResponse.json(
      {
        errors: [
          {
            name: "base",
            reason: `Couldn't add. ${filename} package is already added (same SHA-256 hash).`,
          },
        ],
      },
      { status: 409 }
    );
  }
);

// POST /software/package — 10-package limit rejection. Copy is verbatim.
export const addPackageLimitHandler = (titleName = "Fleet osquery") =>
  http.post(addPackageUrl, () =>
    HttpResponse.json(
      {
        errors: [
          {
            name: "base",
            reason: `Couldn't add. ${titleName} already has ${MAX_PACKAGES_PER_TITLE} packages. Before adding, delete one you no longer use.`,
          },
        ],
      },
      { status: 409 }
    )
  );

// POST /software/package — FMA-conflict rejection (preserved from pre-multi-package
// behavior). Surfaces when the title already has a Fleet-maintained app.
export const addPackageFmaConflictHandler = (
  titleName = "Zoom",
  fleetName = "Testing & QA"
) =>
  http.post(addPackageUrl, () =>
    HttpResponse.json(
      {
        errors: [
          {
            name: "base",
            reason: `Couldn't add. ${titleName} already has a Fleet-maintained app on the ${fleetName} fleet.`,
          },
        ],
      },
      { status: 409 }
    )
  );

// POST /software/package — VPP-conflict rejection (preserved from pre-multi-package
// behavior). Surfaces when the title already has an Apple App Store (VPP) app.
export const addPackageVppConflictHandler = (
  titleName = "Zoom",
  fleetName = "Testing & QA"
) =>
  http.post(addPackageUrl, () =>
    HttpResponse.json(
      {
        errors: [
          {
            name: "base",
            reason: `Couldn't add. ${titleName} already has an Apple App Store (VPP) on the ${fleetName} fleet.`,
          },
        ],
      },
      { status: 409 }
    )
  );

// PATCH /software/titles/:id/package — per-installer edit. Echoes the
// targeted installer_id so tests can assert the request hit the right row.
export const editPackageHandler = http.patch(
  editPackageUrl,
  async ({ request, params }) => {
    const form = await request.formData();
    return HttpResponse.json({
      software_title_id: Number(params.id),
      installer_id: Number(form.get("installer_id") ?? 0),
    });
  }
);

// DELETE /software/titles/:id/available_for_install — per-installer delete.
// 204 when other packages remain; same shape as today when last.
export const deletePackageHandler = http.delete(
  deletePackageUrl,
  () => new HttpResponse(null, { status: 204 })
);

export const deletePackageErrorHandler = http.delete(deletePackageUrl, () =>
  HttpResponse.json(
    { errors: [{ name: "base", reason: "Internal Server Error" }] },
    { status: 500 }
  )
);
