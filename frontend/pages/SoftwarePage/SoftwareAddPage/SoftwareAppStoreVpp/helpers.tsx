import React, { ReactElement } from "react";
import { getErrorReason } from "interfaces/errors";
import { IMdmVppToken } from "interfaces/mdm";

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

const ADD_SOFTWARE_ERROR_PREFIX = "Couldn't add software.";
const DEFAULT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;

const generateAlreadyAvailableMessage = (
  msg: string
): string | ReactElement => {
  // This regex matches the API message where the title already has a software package (non-VPP) available for install.
  const regex = new RegExp(
    `${ADD_SOFTWARE_ERROR_PREFIX} (.+) already.+on the (.+) team.`
  );

  const match = msg.match(regex);

  if (match) {
    return (
      <>
        {ADD_SOFTWARE_ERROR_PREFIX} <b>{match[1]}</b> already has software
        available for install on the <b>{match[2]}</b> team.{" "}
      </>
    );
  }

  if (msg.includes("VPPApp")) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install on this team.`;
  }

  return DEFAULT_ERROR_MESSAGE;
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown): string | ReactElement => {
  let reason = getErrorReason(e);

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    return generateAlreadyAvailableMessage(reason);
  }

  if (reason && !reason.endsWith(".")) {
    reason += ".";
  }

  return reason || DEFAULT_ERROR_MESSAGE;
};
