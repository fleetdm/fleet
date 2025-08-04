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
 * Returns a string of comma-separated team ids from a token.
 * Special handling for "All teams".
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
  const ids = selectedValue
    .split(",")
    .map((str) => parseInt(str, 10))
    .filter((id) => !isNaN(id));
  return ids;
};

/**
 * Compare two comma-separated strings of team ids and returns an updated value.
 * Includes special handling for "All teams".
 */
export const updateSelectedValue = (prev: string, next: string) => {
  // react-select uses a string of comma-separated values for multi-select so we split it
  // fo get an array of selected team ids
  const nextParts = next.split(",").map((p) => p.trim());
  if (nextParts.length === 1) {
    return next;
  }

  const allTeamsId = APP_CONTEXT_ALL_TEAMS_ID.toString();
  const prevParts = prev.split(",").map((p) => p.trim());
  if (prevParts.includes(allTeamsId)) {
    // if "All teams" was previously selected, we need to remove it from the next selections
    return nextParts.filter((p) => p !== allTeamsId).join(",");
  }

  // If "All teams" is newly selected, remove other selections
  if (nextParts.includes(allTeamsId)) {
    return allTeamsId;
  }

  // Otherwise, just return the next selections
  return next;
};

const isTokenAllTeams = (token: IMdmVppToken) => token.teams?.length === 0;
const isTokenUnassigned = (token: IMdmVppToken) => token.teams === null;

/**
 * Returns a dictionary of team ids already assigned (other than current token).
 */
const getUnavailableTeamIds = (
  currentTokenId: number,
  tokens: IMdmVppToken[]
) => {
  const unavailableTeamIds: Record<string, boolean> = {};
  tokens.forEach((token) => {
    if (token.id === currentTokenId) return;
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
  currentToken: IMdmVppToken,
  pendingTeamIds: string[]
) => {
  const allTeamsOption = {
    label: "All teams",
    value: APP_CONTEXT_ALL_TEAMS_ID,
  };

  // If another token already owns "All teams", return empty dropdown
  const anotherTokenIsAllTeams = tokens.some(
    (token) => isTokenAllTeams(token) && token.id !== currentToken.id
  );
  if (anotherTokenIsAllTeams) return [];

  // Filter for actual team options, add "All teams" to the front
  const allOptions = [
    allTeamsOption,
    ...availableTeams
      .filter((t) => t.id !== APP_CONTEXT_ALL_TEAMS_ID)
      .map((t) => ({
        label: t.name,
        value: t.id,
      })),
  ];

  // Determine state of pending assignment
  const isPendingAllTeams = pendingTeamIds?.includes(
    APP_CONTEXT_ALL_TEAMS_ID.toString()
  );

  // Case 1: All tokens are unassigned → show all options, including "All teams"
  if (tokens.every(isTokenUnassigned)) {
    return allOptions;
  }

  // Case 2: If another token (not current) is assigned "All teams", restrict everything unless current/pending choosing "all teams"
  if (
    tokens.some(
      (token) => isTokenAllTeams(token) && token.id !== currentToken.id
    ) &&
    !isPendingAllTeams
  ) {
    return [];
  }

  // Case 3: If ANY other token is assigned real teams (not all teams/not unassigned)...
  const anotherAssigned = tokens
    .filter((t) => t.id !== currentToken.id)
    .some((t) => !isTokenAllTeams(t) && !isTokenUnassigned(t));

  // If so, and we're not actively changing this token to "All teams", REMOVE "All teams" option
  let filteredOptions = allOptions;
  if (anotherAssigned && !isPendingAllTeams) {
    filteredOptions = allOptions.filter(
      (o) => o.value !== APP_CONTEXT_ALL_TEAMS_ID
    );
  }

  // Get teams unavailable due to assignment to other tokens
  const unavailableTeamIds = getUnavailableTeamIds(currentToken.id, tokens);

  // Return options not assigned, or that are in the pending selection
  return filteredOptions.filter(
    (o) =>
      !unavailableTeamIds[o.value.toString()] ||
      pendingTeamIds.includes(o.value.toString())
  );
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

  const selectedValueArr = useMemo(
    () =>
      selectedValue
        ? selectedValue
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean)
        : [],
    [selectedValue]
  );

  const options = useMemo(() => {
    return getOptions(
      availableTeams || [],
      tokens,
      currentToken,
      selectedValueArr
    );
  }, [availableTeams, tokens, currentToken, selectedValueArr]);

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
      } finally {
        setIsSaving(false);
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
          If you delete a team, App Store apps will be deleted from that team.
          Installed apps won&apos;t be uninstalled from hosts.
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
