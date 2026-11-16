import url_prefix from "router/url_prefix";

const deviceSelfServiceRegex = new RegExp(
  `^${url_prefix}/device/[^/]+/self-service/?$`
);

// iOS/iPadOS base device route should support low-width screens
const deviceIOSIPadOSRegex = new RegExp(`^${url_prefix}/device/[^/]+/?$`);

// Device paths render a mobile UI, so narrow viewports are only valid there
// when the viewer is actually on a mobile device (or has forced it via
// ?mobile=1). Elsewhere, narrow desktop always gets the unsupported overlay.
const lowWidthSupportedPathsRegex = [
  deviceSelfServiceRegex,
  deviceIOSIPadOSRegex,
];

const shouldShowUnsupportedScreen = (
  locationPathname: string,
  isMobile = false
) => {
  const isLowWidthSupportedPath = lowWidthSupportedPathsRegex.some((regex) =>
    regex.test(locationPathname)
  );
  return !(isLowWidthSupportedPath && isMobile);
};

export default shouldShowUnsupportedScreen;
