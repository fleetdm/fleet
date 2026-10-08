import { IConfigOverrideParamsOnTeamChange } from "hooks/useTeamIdParam";
import { HostStatusFilter } from "interfaces/host";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { FLEET_SCOPED_HOST_FILTER_PARAMS } from "services/entities/hosts";

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

const isAllFleets = (newTeamId?: number) =>
  newTeamId === APP_CONTEXT_ALL_TEAMS_ID;

export const STRIP_FLEET_SCOPED_FILTERS_ON_ALL_FLEETS = FLEET_SCOPED_HOST_FILTER_PARAMS.reduce<IConfigOverrideParamsOnTeamChange>(
  (config, param) => ({ ...config, [param]: isAllFleets }),
  {}
);
