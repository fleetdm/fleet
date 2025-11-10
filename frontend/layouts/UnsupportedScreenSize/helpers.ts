import url_prefix from "router/url_prefix";

const deviceSelfServiceRegex = new RegExp(
  `^${url_prefix}/device/[^/]+/self-service/?$`
);

// Define paths that will not show the unsupported screen overlay
const lowWidthSupportedPathsRegex = [deviceSelfServiceRegex];

const shouldShowUnsupportedScreen = (locationPathname: string) =>
  !lowWidthSupportedPathsRegex.some((regex) => regex.test(locationPathname));

export default shouldShowUnsupportedScreen;
