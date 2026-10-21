import { isPast } from "date-fns";
import { ReactElement } from "react";

import { getErrorReason } from "interfaces/errors";
import { IMdmVppToken } from "interfaces/mdm";

import {
  ADD_SOFTWARE_ERROR_PREFIX,
  DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE,
  ensurePeriod,
  formatAlreadyAvailableInstallMessage,
} from "../../helpers";

/** Tokens assigned to the fleet, directly or through All fleets, first added first. */
export const getAssignedVppTokens = (
  teamId: number,
  tokens: IMdmVppToken[] = []
) =>
  tokens
    .filter(
      // A non-null, empty `teams` means the token is assigned to All fleets.
      (t) =>
        t.teams?.length === 0 || t.teams?.some((tm) => tm.team_id === teamId)
    )
    .sort((a, b) => a.id - b.id);

/** Assigned tokens that haven't expired. */
export const getFleetVppTokens = (teamId: number, tokens?: IMdmVppToken[]) =>
  getAssignedVppTokens(teamId, tokens).filter(
    (t) => !isPast(new Date(t.renew_date))
  );

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
      return `${ADD_SOFTWARE_ERROR_PREFIX} The software is already available to install in this fleet.`;
    }
  }

  if (reason) {
    return `${ADD_SOFTWARE_ERROR_PREFIX} ${ensurePeriod(reason)}`;
  }

  return DEFAULT_ADD_SOFTWARE_ERROR_MESSAGE;
};
