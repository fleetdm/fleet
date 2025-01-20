import React, { useCallback, useContext, useMemo, useState } from "react";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { IMdmVppToken } from "interfaces/mdm";
import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";

import mdmAppleAPI from "services/entities/mdm_apple";

import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "edit-teams-vpp-modal";

interface IEditTeamsVppModalProps {
  currentToken: IMdmVppToken;
  tokens: IMdmVppToken[];
  onCancel: () => void;
  onSuccess: () => void;
}

/**
 * Returns an array of team ids from a token. It includes special handling for "All teams".
 */
export const selectedValueFromToken = (token: IMdmVppToken) => {
  if (!token.teams) {
    // if teams is null, it means the token isn't configured to any team
    return "";
  }
  if (token.teams.length === 0) {
    // if teams is an empty array, it means the token is configured to all teams
    return APP_CONTEXT_ALL_TEAMS_ID.toString();
  }
  return token.teams.map((team) => team.team_id).join(",");
};

/** Returns an array of team ids for the API. It includes special handling for "All teams". */
export const teamIdsFromSelectedValue = (selectedValue: string) => {
  if (!selectedValue) {
    // TODO: confirm whether the API will allow un-configuring a token
    return null;
  }
  if (selectedValue === APP_CONTEXT_ALL_TEAMS_ID.toString()) {
    return [];
  }
  const ids = selectedValue.split(",").map((str) => parseInt(str, 10));
  // NOTE: We could do some extra frontend validation here like filtering out -1 (to ensure that
  // we're not trying to send all teams and other teams at the same time) and checking for isNaN,
  // but instead we're relying on the API to return an error if the request is invalid.
  return ids;
};

/**
 * Compare two comma-separated strings of team names and returns an updated value. It includes
 * special handling for "All teams".
 */
export const updateSelectedValue = (prev: string, next: string) => {
  // react-select uses a string of comma-separated values for multi-select so we split it
  // fo get an array of selected team ids
  const nextParts = next.split(",").map((p) => p.trim());
  if (nextParts.length === 1) {
    // if only one team is selected, no need for other checks
    return next;
  }

  // we need to do some special handling for "All teams"
  const allTeamsId = APP_CONTEXT_ALL_TEAMS_ID.toString();
  // split the previous value to get an array of team ids
  const prevParts = prev.split(",").map((p) => p.trim());
  if (prevParts.includes(allTeamsId)) {
    // if "All teams" was previously selected, we need to remove it from the next selections
    return nextParts.filter((p) => p !== allTeamsId).join(", ");
  }

  // if "All teams" is newly selected, we need to remove any other selections
  if (nextParts.includes(allTeamsId)) {
    return allTeamsId;
  }

  // otherwise, just return the next selections
  return next;
};

const isTokenAllTeams = (token: IMdmVppToken) => token.teams?.length === 0;

const isTokenUnassigned = (token: IMdmVppToken) => token.teams === null;

/**
 * Returns a dictionary of team ids that are already assigned tokens other than the current token.
 */
const getUnavailableTeamIds = (
  currentTokenId: number,
  tokens: IMdmVppToken[]
) => {
  const unavailableTeamIds = {} as Record<string, boolean>;
  tokens.forEach((token) => {
    if (token.id === currentTokenId) {
      return;
    }
    token.teams?.forEach((team) => {
      unavailableTeamIds[team.team_id.toString()] = true;
    });
  });
  return unavailableTeamIds;
};

/**
 * Returns an array of options for the dropdown. It includes special handling for "All teams".
 */
export const getOptions = (
  availableTeams: ITeamSummary[],
  tokens: IMdmVppToken[],
  currentToken: IMdmVppToken
) => {
  const allOptions =
    availableTeams?.map((t) => ({
      label: t.name,
      value: t.id,
    })) || [];

  if (tokens.every(isTokenUnassigned)) {
    // if all tokens are unassigned, we can include all options
    return allOptions;
  }

  if (tokens.some(isTokenAllTeams) && !isTokenAllTeams(currentToken)) {
    // if another token is assigned to all teams, we can't assign this token to any team
    return [];
  }

  // if other tokens are assigned to specific teams, we'll filter out those team options
  const unavailableTeamIds = getUnavailableTeamIds(currentToken.id, tokens);
  if (!isTokenAllTeams(currentToken)) {
    // if current token isn't already assigned to all teams, we'll exclude that option too
    unavailableTeamIds[APP_CONTEXT_ALL_TEAMS_ID] = true;
  }

  return allOptions.filter((o) => !unavailableTeamIds[o.value]);
};

const EditTeamsVppModal = ({
  tokens,
  currentToken,
  onCancel,
  onSuccess,
}: IEditTeamsVppModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { availableTeams } = useContext(AppContext);

  // react-select uses a string of comma-separated values for multi-select so we're using a string
  // of ids here so that we don't need to worry about a team name including a comma
  const [selectedValue, setSelectedValue] = useState<string>(
    selectedValueFromToken(currentToken)
  );
  const [isSaving, setIsSaving] = useState(false);

  const options = useMemo(() => {
    return getOptions(availableTeams || [], tokens, currentToken);
  }, [availableTeams, tokens, currentToken]);

  const isAnyTokenAllTeams = useMemo(() => tokens.some(isTokenAllTeams), [
    tokens,
  ]);

  const onChange = useCallback((val: string) => {
    setSelectedValue((prev) => updateSelectedValue(prev, val));
  }, []);

  const onSave = useCallback(
    async (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();
      setIsSaving(true);
      try {
        await mdmAppleAPI.editVppTeams({
          tokenId: currentToken.id,
          teamIds: teamIdsFromSelectedValue(selectedValue),
        });
        renderFlash("success", "Edited successfully.");
        onSuccess();
      } catch (e) {
        renderFlash("error", "Couldn’t edit. Please try again.");
      }
    },
    [currentToken.id, selectedValue, renderFlash, onSuccess]
  );

  const isDropdownDisabled = options.length === 0 && isAnyTokenAllTeams;

  return (
    <Modal
      className={baseClass}
      title="Edit teams"
      onExit={onCancel}
      width="large"
      isContentDisabled={isSaving}
    >
      <>
        <p>
          Edit teams for <b>{currentToken.org_name}</b>.
        </p>
        <p>
          If you delete a team, App Store apps will be deleted from that team,
          and policies that trigger automatic install of these apps will be
          removed. Installed apps won&apos;t be uninstalled from hosts.
        </p>
        <form onSubmit={onSave} className={baseClass} autoComplete="off">
          <TooltipWrapper
            position="top"
            underline={false}
            showArrow
            tipContent={
              <div className={`${baseClass}__tooltip--all-teams`}>
                You can&apos;t choose teams because you already have a VPP token
                assigned to all teams. First, edit teams for that VPP token to
                choose teams here.
              </div>
            }
            disableTooltip={!isDropdownDisabled}
          >
            <Dropdown
              options={options}
              multi
              onChange={onChange}
              placeholder="Search teams"
              value={selectedValue}
              label="Teams"
              className={`${baseClass}__vpp-dropdown`}
              wrapperClassName={`${baseClass}__form-field--vpp-teams ${
                isDropdownDisabled ? `${baseClass}__form-field--disabled` : ""
              }`}
              tooltip={
                isDropdownDisabled ? undefined : (
                  <>
                    Each team can have only one VPP token. Teams that already
                    have a VPP token won&apos;t show up here.
                  </>
                )
              }
              helpText="App Store apps in this VPP token’s Apple Business Manager (ABM) will only be available to install on hosts in these teams."
              disabled={isDropdownDisabled}
            />
          </TooltipWrapper>
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              className="save-vpp-teams-loading"
              isLoading={isSaving}
              disabled={isDropdownDisabled}
            >
              Save
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default EditTeamsVppModal;
