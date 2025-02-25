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

const baseClass = "edit-teams-abm-modal";

interface IEditTeamsAbmModalProps {
  token: IMdmAbmToken;
  onCancel: () => void;
  onSuccess: () => void;
}

/**
 * Given available teams, return the options for the dropdowns. The "All teams" option is excluded.
 */
export const getOptions = (availableTeams: ITeamSummary[] = []) => {
  return availableTeams
    ?.filter((t) => t.name !== "All teams")
    .map((t) => ({
      value: t.name,
      label: t.name,
    }));
};

/**
 * Type for the selected team names, which is derived from the shape of the teams object
 * returned by the get token API.
 */
interface SelectedTeamNames {
  ios_team: IMdmAbmToken["ios_team"]["name"];
  ipados_team: IMdmAbmToken["ipados_team"]["name"];
  macos_team: IMdmAbmToken["macos_team"]["name"];
}

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
export const getSelectedTeamIds = (
  { ios_team, ipados_team, macos_team }: SelectedTeamNames,
  availableTeams: ITeamSummary[] = []
): SelectedTeamIds => {
  const byName = availableTeams.reduce((acc, t) => {
    acc[t.name] = t.id;
    return acc;
  }, {} as Record<string, number>);
  return {
    ios_team_id: byName[ios_team],
    ipados_team_id: byName[ipados_team],
    macos_team_id: byName[macos_team],
  };
};

const EditTeamsAbmModal = ({
  token,
  onCancel,
  onSuccess,
}: IEditTeamsAbmModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { availableTeams } = useContext(AppContext);

  const [isSaving, setIsSaving] = useState(false);

  const [selectedTeamNames, setSelectedTeamNames] = useState<SelectedTeamNames>(
    {
      ios_team: token.ios_team.name,
      ipados_team: token.ipados_team.name,
      macos_team: token.macos_team.name,
    }
  );

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

      setIsSaving(true);
      try {
        await mdmAbmAPI.editTeams({
          tokenId: token.id,
          teams: getSelectedTeamIds(selectedTeamNames, availableTeams),
        });
        renderFlash("success", "Edited successfully.");
        onSuccess();
      } catch (e) {
        renderFlash("error", "Couldn’t edit. Please try again.");
        onCancel();
      }
    },
    [
      token.id,
      selectedTeamNames,
      availableTeams,
      renderFlash,
      onSuccess,
      onCancel,
    ]
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
            tooltip={
              <>
                macOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
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
            tooltip={
              <>
                iOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
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
            tooltip={
              <>
                iPadOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
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
