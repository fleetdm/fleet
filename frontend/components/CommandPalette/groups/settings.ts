import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildSettingsItems = (
  ctx: ICommandPaletteContext,
  _derived: IDerivedContext
): ICommandItem[] => {
  const { canAccessSettings, isPremiumTier, isPrimoMode } = ctx;

  // Settings — global admins only
  if (!canAccessSettings) return [];

  return [
    // Organization settings
    {
      id: "settings-organization",
      label: "Organization settings",
      group: "Settings" as const,
      path: paths.ADMIN_ORGANIZATION,
      keywords: ["admin", "organization"],
      subItems: [
        {
          id: "settings-org-info",
          label: "Organization info",
          path: paths.ADMIN_ORGANIZATION_INFO,
          keywords: ["name", "logo", "branding", "support url"],
        },
        {
          id: "settings-org-webaddress",
          label: "Fleet web address",
          path: paths.ADMIN_ORGANIZATION_WEBADDRESS,
          keywords: ["url", "server address"],
        },
        {
          id: "settings-org-smtp",
          label: "SMTP options",
          path: paths.ADMIN_ORGANIZATION_SMTP,
          keywords: ["email", "sender", "password reset"],
        },
        {
          id: "settings-org-agents",
          label: "Agent options",
          path: paths.ADMIN_ORGANIZATION_AGENTS,
          keywords: [
            "osquery",
            "fleetd",
            "orbit",
            "flags",
            "global config",
            "command line flags",
          ],
        },
        {
          id: "settings-org-statistics",
          label: "Usage statistics",
          path: paths.ADMIN_ORGANIZATION_STATISTICS,
          keywords: ["telemetry", "anonymous"],
        },
        {
          id: "settings-org-fleet-desktop",
          label: "Fleet Desktop",
          path: paths.ADMIN_ORGANIZATION_FLEET_DESKTOP,
          keywords: [
            "tray icon",
            "transparency",
            "end user",
            "browser host",
            "custom proxy",
          ],
        },
        {
          id: "settings-org-advanced",
          label: "Advanced options",
          path: paths.ADMIN_ORGANIZATION_ADVANCED,
          keywords: [
            "live report",
            "host expiry",
            "usage statistics",
            "sso user url",
            "sso",
            "apple mdm server url",
            "verify ssl certs",
            "starttls",
            "host expiry",
            "generative ai features",
            "hardware attestation",
          ],
        },
      ],
    },

    // Integrations
    {
      id: "settings-integrations",
      label: "Integrations",
      group: "Settings" as const,
      path: paths.ADMIN_INTEGRATIONS,
      keywords: ["mdm", "jira", "zendesk", "sso", "calendar"],
      subItems: [
        {
          id: "settings-int-ticket-destinations",
          label: "Ticket destinations",
          path: paths.ADMIN_INTEGRATIONS_TICKET_DESTINATIONS,
          keywords: ["jira", "zendesk", "tickets"],
        },
        {
          id: "settings-int-mdm",
          label: "MDM",
          path: paths.ADMIN_INTEGRATIONS_MDM,
          keywords: [
            "apple",
            "windows",
            "android",
            "device management",
            "apple business",
            "vpp",
            "entra",
          ],
        },
        // Calendars and Change management are Premium-only.
        ...(isPremiumTier
          ? [
              {
                id: "settings-int-calendars",
                label: "Calendars",
                path: paths.ADMIN_INTEGRATIONS_CALENDARS,
                keywords: [
                  "google calendar api",
                  "google workspace",
                  "service account",
                  "events",
                ],
              },
              {
                id: "settings-int-change-management",
                label: "Change management",
                path: paths.ADMIN_INTEGRATIONS_CHANGE_MANAGEMENT,
                keywords: ["workflow", "gitops mode"],
              },
            ]
          : []),
        {
          id: "settings-int-sso-fleet-users",
          label: "Single sign-on (SSO) for Fleet users",
          path: paths.ADMIN_INTEGRATIONS_SSO_FLEET_USERS,
          keywords: ["saml", "idp", "admin", "login"],
        },
        {
          id: "settings-int-sso-end-users",
          label: "Single sign-on (SSO) for end users",
          path: paths.ADMIN_INTEGRATIONS_SSO_END_USERS,
          keywords: ["saml", "idp", "device user", "login"],
        },
        // Certificate authorities pages are Premium-only.
        ...(isPremiumTier
          ? [
              {
                id: "settings-int-certificate-authorities",
                label: "Certificate authorities",
                path: paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
                keywords: [
                  "scep",
                  "est",
                  "digicert",
                  "ndes",
                  "smallstep",
                  "scep",
                ],
              },
              {
                id: "add-certificate-authority",
                label: "Add certificate authority",
                path: paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
                keywords: [
                  "scep",
                  "est",
                  "digicert",
                  "ndes",
                  "smallstep",
                  "pki",
                ],
              },
            ]
          : []),
        {
          id: "settings-int-identity-provider",
          label: "Identity provider (IdP)",
          path: paths.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER,
          keywords: ["okta", "entra", "azure ad"],
        },
        {
          id: "settings-int-host-status-webhook",
          label: "Host status webhook",
          path: paths.ADMIN_INTEGRATIONS_HOST_STATUS_WEBHOOK,
          keywords: ["offline", "missing hosts", "notification"],
        },
        // Conditional access is Premium-only.
        ...(isPremiumTier
          ? [
              {
                id: "settings-int-conditional-access",
                label: "Conditional access",
                path: paths.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS,
                keywords: ["okta", "entra", "intune", "zero trust"],
              },
            ]
          : []),
      ],
    },

    // Users and Fleets
    {
      id: "settings-users",
      label: "Users",
      group: "Settings" as const,
      path: paths.ADMIN_USERS,
      keywords: ["accounts", "admins", "invite"],
    },
    // Fleets settings tab — Premium-only, and hidden in Primo Mode
    // (single-fleet installs don't expose fleet management).
    ...(isPremiumTier && !isPrimoMode
      ? [
          {
            id: "settings-fleets",
            label: "Fleets",
            group: "Settings" as const,
            path: paths.ADMIN_FLEETS,
            keywords: [
              "teams",
              "groups",
              "add fleet",
              "create fleet",
              "edit fleet",
              "delete fleet",
            ],
          },
        ]
      : []),
  ];
};

export default buildSettingsItems;
