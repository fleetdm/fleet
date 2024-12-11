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
