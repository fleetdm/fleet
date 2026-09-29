import { IConfigOverrideParamsOnTeamChange } from "hooks/useTeamIdParam";
import { HostStatusFilter } from "interfaces/host";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";

export const isAcceptableStatus = (
  filter?: string
): filter is HostStatusFilter => {
  return (
    filter === "new" ||
    filter === "online" ||
    filter === "offline" ||
    filter === "missing" ||
    filter === "pending" ||
    filter === "enrolled"
  );
};

export const isValidPolicyResponse = (filter: string): boolean => {
  return filter === "pass" || filter === "fail";
};

// Performs a grossly oversimplied validation that subject string includes substrings
// that would be expected in a textual encoding of a certificate chain per the PEM spec
// (see https://datatracker.ietf.org/doc/html/rfc7468#section-2)
// Consider using a third-party library if more robust validation is desired
export const isValidPemCertificate = (cert: string): boolean => {
  const regexPemHeader = /-----BEGIN/;
  const regexPemFooter = /-----END/;

  return regexPemHeader.test(cert) && regexPemFooter.test(cert);
};

// These filters only apply within one fleet or "No fleet": without a fleet the
// API narrows them to "No fleet", which would show under an "All fleets" label.
const isAllFleets = (newTeamId?: number) =>
  newTeamId === APP_CONTEXT_ALL_TEAMS_ID;

export const STRIP_FLEET_SCOPED_FILTERS_ON_ALL_FLEETS: IConfigOverrideParamsOnTeamChange = {
  [HOSTS_QUERY_PARAMS.OS_SETTINGS]: isAllFleets,
  apple_settings: isAllFleets,
  macos_settings: isAllFleets,
  [HOSTS_QUERY_PARAMS.DISK_ENCRYPTION]: isAllFleets,
  macos_bootstrap_package: isAllFleets,
  bootstrap_package: isAllFleets,
  [HOSTS_QUERY_PARAMS.SOFTWARE_STATUS]: isAllFleets,
};
