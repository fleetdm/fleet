import { isAxiosError } from "axios";
import { getErrorReason } from "interfaces/errors";

import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  REQUEST_TIMEOUT_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../../helpers";
import fleetAppData from "../../../../../../server/mdm/maintainedapps/apps.json";

const NameToIdentifierMap: Record<string, string> = {
  "1Password": "1password",
  "Adobe Acrobat Reader": "adobe-acrobat-reader",
  "Box Drive": "box-drive",
  Brave: "brave-browser",
  "Cloudflare WARP": "cloudflare-warp",
  "Docker Desktop": "docker",
  Figma: "figma",
  "Mozilla Firefox": "firefox",
  "Google Chrome": "google-chrome",
  "Microsoft Edge": "microsoft-edge",
  "Microsoft Excel": "microsoft-excel",
  "Microsoft Teams": "microsoft-teams",
  "Microsoft Word": "microsoft-word",
  Notion: "notion",
  Postman: "postman",
  Slack: "slack",
  TeamViewer: "teamviewer",
  "Microsoft Visual Studio Code": "visual-studio-code",
  WhatsApp: "whatsapp",
  Zoom: "zoom",
  "Zoom for IT Admins": "zoom-for-it-admins",
};

const getFleetAppData = (name: string) => {
  const appId = NameToIdentifierMap[name]; // TODO: need a better matching mechanism here
  return fleetAppData.find((app) => app.identifier === appId);
};

export const getFleetAppPolicyName = (appName: string) => {
  return `[Install software] ${appName}`;
};

export const getFleetAppPolicyDescription = (appName: string) => {
  return `Policy triggers automatic install of ${appName} on each host that's missing this software.`;
};

export const getFleetAppPolicyQuery = (name: string) => {
  return getFleetAppData(name)?.automatic_policy_query;
};

export const getErrorMessage = (err: unknown) => {
  const isTimeout =
    isAxiosError(err) &&
    (err.response?.status === 504 || err.response?.status === 408);
  const reason = getErrorReason(err);

  if (
    isTimeout ||
    reason.includes("json decoder error") // 400 bad request when really slow
  ) {
    return REQUEST_TIMEOUT_ERROR_MESSAGE;
  }

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    const alreadyAvailableMessage = formatAlreadyAvailableInstallMessage(
      reason
    );
    if (alreadyAvailableMessage) {
      return alreadyAvailableMessage;
    }
  }

  if (reason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  }
  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};
