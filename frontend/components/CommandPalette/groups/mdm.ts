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
            keywords: [
              "enable",
              "activate",
              "set up apple mdm",
              "configure apple mdm",
              "apns",
              "dep",
              "iphone",
              "ipad",
              "macbook",
            ],
          },
        ]
      : [
          {
            id: "edit-apple-mdm",
            label: "Edit Apple (macOS, iOS, iPadOS) MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
            keywords: [
              "update apple mdm",
              "change apple mdm",
              "modify apple mdm",
              "configure apple mdm",
              "apns",
              "certificate",
              "renew",
              "iphone",
              "ipad",
              "macbook",
            ],
          },
          // AB and VPP pages are Premium-only.
          ...(isPremiumTier
            ? [
                {
                  id: isAbmConfigured ? "edit-abm" : "add-abm",
                  label: isAbmConfigured
                    ? "Edit Apple Business (AB)"
                    : "Add Apple Business (AB)",
                  group: "MDM" as const,
                  path: paths.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER,
                  keywords: [
                    "dep",
                    "automated enrollment",
                    "apple",
                    ...(isAbmConfigured
                      ? ["update abm", "change abm", "modify abm", "configure"]
                      : ["create abm", "new abm", "configure", "set up abm"]),
                  ],
                },
                {
                  id: isVppEnabled ? "edit-vpp" : "add-vpp",
                  label: isVppEnabled
                    ? "Edit Volume Purchasing Program (VPP)"
                    : "Add Volume Purchasing Program (VPP)",
                  group: "MDM" as const,
                  path: paths.ADMIN_INTEGRATIONS_VPP,
                  keywords: [
                    "app store",
                    "apple",
                    "token",
                    ...(isVppEnabled
                      ? ["update vpp", "change vpp", "modify vpp", "configure"]
                      : ["create vpp", "new vpp", "configure", "set up vpp"]),
                  ],
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
            keywords: [
              "enable",
              "activate",
              "set up windows mdm",
              "configure windows mdm",
              "microsoft",
              "pc",
              "win10",
              "win11",
            ],
          },
        ]
      : [
          {
            id: "edit-windows-mdm",
            label: "Edit Windows MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
            keywords: [
              "update windows mdm",
              "change windows mdm",
              "modify windows mdm",
              "configure windows mdm",
              "microsoft",
              "enrollment",
              "pc",
              "win10",
              "win11",
            ],
          },
          {
            id: "windows-automatic-enrollment",
            label: "Windows automatic enrollment (Entra)",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS,
            keywords: [
              "entra",
              "azure ad",
              "microsoft",
              "autopilot",
              "active directory",
            ],
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
            keywords: [
              "enable",
              "activate",
              "set up android mdm",
              "configure android mdm",
              "google",
              "enterprise",
              "phone",
              "tablet",
            ],
          },
        ]
      : [
          {
            id: "edit-android-mdm",
            label: "Edit Android MDM",
            group: "MDM" as const,
            path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
            keywords: [
              "update android mdm",
              "change android mdm",
              "modify android mdm",
              "configure android mdm",
              "google",
              "enterprise",
              "phone",
              "tablet",
            ],
          },
        ]),
  ];
};

export default buildMdmItems;
