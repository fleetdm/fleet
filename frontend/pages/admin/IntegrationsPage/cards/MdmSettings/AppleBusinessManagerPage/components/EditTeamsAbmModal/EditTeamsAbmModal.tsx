import React, { useCallback, useContext, useMemo, useState } from "react";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { IMdmAbmToken } from "interfaces/mdm";
import { ITeamSummary } from "interfaces/team";

import mdmAbmAPI from "services/entities/mdm_apple_bm";

import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";

const baseClass = "add-abm-modal";

interface IEditTeamsAbmModalProps {
  token: IMdmAbmToken;
  onCancel: () => void;
  onSuccess: () => void;
}

/**
 * Type for the selected team names, which takes the shape of the teams object
 * returned by the get token API.
 */
type SelectedTeamNames = Pick<
  IMdmAbmToken,
  "macos_team" | "ios_team" | "ipados_team"
>;

/**
 * Type for the selected team ids, which takes the shape of the teams object
 * expected by the edit token teams API.
 */
type SelectedTeamIds = Parameters<typeof mdmAbmAPI.editTeams>[0]["teams"];

/**
 * Given selected team names and available teams, return the corresponding team ids with the keys
 * expected by the API. If a team name does not exist in the available teams, its id will be undefined.
 * The caller is responsible for handling undefined ids appropriately (e.g., using error messages from
 * `validateSelectedTeamIds` function).
 */
const getSelectedTeamIds = (
  { ios_team, ipados_team, macos_team }: SelectedTeamNames,
  availableTeams: ITeamSummary[]
) => {
  const byName = availableTeams.reduce((acc, t) => {
    acc[t.name] = t.id;
    return acc;
  }, {} as Record<string, number | undefined>);
  return {
    ios_team_id: byName[ios_team],
    ipados_team_id: byName[ipados_team],
    macos_team_id: byName[macos_team],
  };
};

/**
 * Validate that the selected team names have valid team ids. If any team id is undefined, return an
 * error message indicating which team is invalid.
 *
 * Note: Ideally, we shouldn't need to validate that team names have valid team ids if the backend
 * is adequately protecting against "broken" teams (where users change the name of a team but we
 * forget to update stored references to the old name). This is included as a safeguard and to
 * provide a "noisier" error message to surface potential backend issues.
 */
const validateSelectedTeamIds = (
  { ios_team_id, ipados_team_id, macos_team_id }: SelectedTeamIds,
  { ios_team, ipados_team, macos_team }: SelectedTeamNames
) => {
  if (ios_team_id === undefined) {
    return `Selected iOS team ${ios_team} does not have a valid team id.`;
  }
  if (ipados_team_id === undefined) {
    return `Selected iPadOS team ${ipados_team} does not have a valid team id.`;
  }
  if (macos_team_id === undefined) {
    return `Selected macOS team ${macos_team} does not have a valid team id.`;
  }
  return "";
};

const EditTeamsAbmModal = ({
  token,
  onCancel,
  onSuccess,
}: IEditTeamsAbmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { availableTeams } = useContext(AppContext);

  const [isSaving, setIsSaving] = useState(false);

  const [selectedTeamNames, setSelectedTeamNames] = useState({
    ios_team: token.ios_team,
    ipados_team: token.ipados_team,
    macos_team: token.macos_team,
  });

  const options = useMemo(() => {
    return availableTeams
      ?.filter((t) => t.name !== "All teams")
      .map((t) => ({
        value: t.name,
        label: t.name,
      }));
  }, [availableTeams]);

  const onSave = useCallback(
    async (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();
      console.log("onSave", selectedTeamNames, evt);

      const teamIds = getSelectedTeamIds(
        selectedTeamNames,
        availableTeams || []
      );
      const invalidTeamIdErr = validateSelectedTeamIds(
        teamIds as SelectedTeamIds,
        selectedTeamNames
      );
      if (invalidTeamIdErr) {
        renderFlash("error", invalidTeamIdErr);
        return;
      }

      setIsSaving(true);
      console.log("isSaving");
      try {
        await mdmAbmAPI.editTeams({
          tokenId: token.id,
          teams: teamIds as SelectedTeamIds,
        });
        renderFlash("success", "Edited successfully.");
        onSuccess();
      } catch (e) {
        renderFlash("error", "Couldn’t edit. Please try again.");
      }
    },
    [selectedTeamNames, availableTeams, renderFlash, token.id, onSuccess]
  );

  return (
    <Modal
      className={baseClass}
      title="Edit teams"
      onExit={onCancel}
      width="large"
      isContentDisabled={isSaving}
    >
      <>
        {" "}
        <p>
          Edit teams for <b>{token.org_name}</b>.
        </p>
        <form onSubmit={onSave} className={baseClass} autoComplete="off">
          <Dropdown
            searchable={false}
            options={options}
            onChange={(value: string) => {
              setSelectedTeamNames((prev) => ({ ...prev, macos_team: value }));
            }}
            value={selectedTeamNames.macos_team}
            label="macOS team"
            wrapperClassName={`${baseClass}__form-field form-field--macos`}
          />
          <Dropdown
            searchable={false}
            options={options}
            onChange={(value: string) => {
              setSelectedTeamNames((prev) => ({ ...prev, ios_team: value }));
            }}
            value={selectedTeamNames.ios_team}
            label="iOS team"
            wrapperClassName={`${baseClass}__form-field form-field--ios`}
          />
          <Dropdown
            searchable={false}
            options={options}
            onChange={(value: string) =>
              setSelectedTeamNames((prev) => ({ ...prev, ipados_team: value }))
            }
            value={selectedTeamNames.ipados_team}
            label="iPadOS team"
            wrapperClassName={`${baseClass}__form-field form-field--ipados`}
          />
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              className="save-abm-teams-loading"
              isLoading={isSaving}
            >
              Save
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default EditTeamsAbmModal;
