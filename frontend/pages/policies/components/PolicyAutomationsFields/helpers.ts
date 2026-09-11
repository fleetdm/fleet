import { IMdmProfile, ProfilePlatform } from "interfaces/mdm";

// eslint-disable-next-line import/prefer-default-export
export const rewriteProfilePlatform = (platform: ProfilePlatform) => {
  switch (platform) {
    case "darwin":
      return "macOS";
    case "windows":
      return "Windows";
    default:
      // Should not happen, but we guard against it and make it clear it's unsupported.
      return "Unsupported";
  }
};

export const VALID_PROFILE_PLATFORMS: ProfilePlatform[] = ["darwin", "windows"];

export const filterValidProfiles = (val: IMdmProfile): boolean => {
  if (!VALID_PROFILE_PLATFORMS.includes(val.platform)) {
    return false;
  }

  // Apple declarations have platform "darwin", but is prefixed with "d" in the profile UUID
  if (val.profile_uuid.startsWith("d")) {
    return false;
  }

  return true;
};
