import { ReactElement } from "react";
import { getErrorReason } from "interfaces/errors";
import { IMdmVppToken } from "interfaces/mdm";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../helpers";

/**
 * Checks if the given team has an available VPP token (either a token
 * that's associated with the team, or a token that's available to "All
 * teams")
 */
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

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown): string | ReactElement => {
  const reason = getErrorReason(e);

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    const alreadyAvailableMessage = formatAlreadyAvailableInstallMessage(
      reason
    );
    if (alreadyAvailableMessage) {
      return alreadyAvailableMessage;
    }

    if (reason.includes("VPPApp")) {
      return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install on this team.`;
    }
  }

  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};
