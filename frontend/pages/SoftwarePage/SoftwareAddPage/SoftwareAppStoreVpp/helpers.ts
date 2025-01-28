import { IMdmVppToken } from "interfaces/mdm";

/**
 * Checks if the given team has an available VPP token (either a token
 * that's associated with the team, or a token that's available to "All
 * teams")
 */
// eslint-disable-next-line import/prefer-default-export
export const teamHasVPPToken = (
  currentTeamId: number,
  tokens?: IMdmVppToken[]
) => {
  if (!tokens || tokens.length === 0) {
    return false;
  }

  return tokens.some((token) => {
    // if we've got a non-null, empty array it means the token is available for
    // "All teams"
    if (token.teams?.length === 0) {
      return true;
    }

    return token.teams?.some((team) => team.team_id === currentTeamId);
  });
};
