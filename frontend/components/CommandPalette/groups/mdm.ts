import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildMdmItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const {
    canAccessSettings,
    isPremiumTier,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
    isVppEnabled,
  } = ctx;
  const { isAbmConfigured } = derived;

  // MDM section is global-admin only.
  if (!canAccessSettings) return [];

  return [
    // Apple MDM — turn on or edit
    ...(!isMacMdmEnabledAndConfigured
      ? [
          {
            id: "turn-on-apple-mdm",
            label: "Turn on Apple (macOS, iOS, iPadOS) MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
            keywords: ["enable", "apns", "dep"],
          },
        ]
      : [
          {
            id: "edit-apple-mdm",
            label: "Edit Apple (macOS, iOS, iPadOS) MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
            keywords: ["apns", "certificate", "renew"],
          },
          // ABM and VPP pages are Premium-only.
          ...(isPremiumTier
            ? [
                {
                  id: isAbmConfigured ? "edit-abm" : "add-abm",
                  label: isAbmConfigured
                    ? "Edit Apple Business Manager (ABM)"
                    : "Add Apple Business Manager (ABM)",
                  group: "MDM" as const,
                  path: paths.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER,
                  keywords: ["dep", "automated enrollment", "apple"],
                },
                {
                  id: isVppEnabled ? "edit-vpp" : "add-vpp",
                  label: isVppEnabled
                    ? "Edit Volume Purchasing Program (VPP)"
                    : "Add Volume Purchasing Program (VPP)",
                  group: "MDM" as const,
                  path: paths.ADMIN_INTEGRATIONS_VPP,
                  keywords: ["app store", "apple", "token"],
                },
              ]
            : []),
        ]),
    // Windows MDM — turn on or edit
    ...(!isWindowsMdmEnabledAndConfigured
      ? [
          {
            id: "turn-on-windows-mdm",
            label: "Turn on Windows MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
            keywords: ["enable", "microsoft"],
          },
        ]
      : [
          {
            id: "edit-windows-mdm",
            label: "Edit Windows MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
            keywords: ["microsoft", "enrollment"],
          },
          {
            id: "windows-automatic-enrollment",
            label: "Windows automatic enrollment (Entra)",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS,
            keywords: ["entra", "azure ad", "microsoft"],
          },
        ]),
    // Android MDM — turn on or edit
    ...(!isAndroidMdmEnabledAndConfigured
      ? [
          {
            id: "turn-on-android-mdm",
            label: "Turn on Android MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
            keywords: ["enable", "google", "enterprise"],
          },
        ]
      : [
          {
            id: "edit-android-mdm",
            label: "Edit Android MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
            keywords: ["google", "enterprise"],
          },
        ]),
  ];
};

export default buildMdmItems;
