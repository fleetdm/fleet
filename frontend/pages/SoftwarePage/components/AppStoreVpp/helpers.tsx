import React from "react";
import { getErrorReason } from "interfaces/errors";
import { IMdmVppToken } from "interfaces/mdm";
import { IVppApp } from "services/entities/mdm_apple";
import { buildQueryStringFromParams } from "utilities/url";

const ADD_SOFTWARE_ERROR_PREFIX = "Couldn’t add software.";
const DEFAULT_ERROR_MESSAGE = `${ADD_SOFTWARE_ERROR_PREFIX} Please try again.`;

const generateAlreadyAvailableMessage = (msg: string) => {
  // This regex matches the API message where the title already has a software package (non-VPP) available for install.
  const regex = new RegExp(
    `${ADD_SOFTWARE_ERROR_PREFIX} (.+) already.+on the (.+) team.`
  );

  const match = msg.match(regex);
  if (!match) {
    if (msg.includes("VPPApp")) {
      // This is the case where someone already added this VPP app. This should almost never happen
      // because we omit apps that are already available from the list in the UI, but just in case of
      // shenanigans with concurrent requests or something, we'll handle it with a generic message.
      // The list should clear itself up on the next page load.
      return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install on this team.`;
    }
    return DEFAULT_ERROR_MESSAGE;
  }

  return (
    <>
      {ADD_SOFTWARE_ERROR_PREFIX} <b>{match[1]}</b> already has software
      available for install on the <b>{match[2]}</b> team.{" "}
    </>
  );
};

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  const reason = getErrorReason(e);

  // software is already available for install
  if (reason.toLowerCase().includes("already")) {
    return generateAlreadyAvailableMessage(reason);
  }
  return DEFAULT_ERROR_MESSAGE;
};

export const getUniqueAppId = (app: IVppApp) =>
  `${app.app_store_id}_${app.platform}`;

/**
 * Generates the query params for the redirect to the software page. This
 * will either generate query params to filter by available for install or
 * self service.
 */
export const generateRedirectQueryParams = (
  teamId: number,
  isSelfService: boolean
) => {
  let queryParams = buildQueryStringFromParams({ team_id: teamId });
  if (isSelfService) {
    queryParams = `${queryParams}&self_service=true`;
  } else {
    queryParams = `${queryParams}&available_for_install=true`;
  }
  return queryParams;
};

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
