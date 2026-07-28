import React, { useCallback, useContext, useMemo, useState } from "react";

import { AppContext } from "context/app";

import { IMdmAbToken } from "interfaces/mdm";
import { ITeamSummary } from "interfaces/team";

import mdmAbmAPI from "services/entities/mdm_apple_bm";

import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import { notify } from "components/ToastNotification";
import FormField from "components/forms/FormField";
import RenewDateCell from "../../../components/RenewDateCell";

const baseClass = "edit-teams-abm-modal";

interface IEditTeamsAbmModalProps {
  token: IMdmAbToken;
  onCancel: () => void;
  onSuccess: () => void;
}

/**
 * Given available teams, return the options for the dropdowns. The "All teams" option is excluded.
 */
export const getOptions = (availableTeams: ITeamSummary[] = []) => {
  return availableTeams
    ?.filter((t) => t.name !== "All fleets")
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
  ios_team: IMdmAbToken["ios_fleet"]["name"];
  ipados_team: IMdmAbToken["ipados_fleet"]["name"];
  macos_team: IMdmAbToken["macos_fleet"]["name"];
  byod_team: IMdmAbToken["byod_fleet"]["name"];
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
  { ios_team, ipados_team, macos_team, byod_team }: SelectedTeamNames,
  availableTeams: ITeamSummary[] = []
): SelectedTeamIds => {
  const byName = availableTeams.reduce((acc, t) => {
    acc[t.name] = t.id;
    return acc;
  }, {} as Record<string, number>);
  return {
    ios_fleet_id: byName[ios_team],
    ipados_fleet_id: byName[ipados_team],
    macos_fleet_id: byName[macos_team],
    byod_fleet_id: byName[byod_team],
  };
};

const EditTeamsAbmModal = ({
  token,
  onCancel,
  onSuccess,
}: IEditTeamsAbmModalProps) => {
  const { availableTeams } = useContext(AppContext);

  const [isSaving, setIsSaving] = useState(false);

  const [selectedTeamNames, setSelectedTeamNames] = useState<SelectedTeamNames>(
    {
      ios_team: token.ios_fleet.name,
      ipados_team: token.ipados_fleet.name,
      macos_team: token.macos_fleet.name,
      byod_team: token.byod_fleet.name,
    }
  );

  const options = useMemo(() => {
    return availableTeams
      ?.filter((t) => t.name !== "All fleets")
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
        notify.success(`Successfully updated fleets for ${token.org_name}`);
        onSuccess();
      } catch (e) {
        notify.error("Couldn’t edit. Please try again.", { response: e });
        onCancel();
      }
    },
    [
      token.id,
      token.org_name,
      selectedTeamNames,
      availableTeams,
      onSuccess,
      onCancel,
    ]
  );

  return (
    <Modal
      className={baseClass}
      title={token.org_name}
      onExit={onCancel}
      width="large"
      isContentDisabled={isSaving}
    >
      <form onSubmit={onSave} className={baseClass} autoComplete="off">
        <FormField name="apple_id" label="Apple ID">
          <p>{token.apple_id}</p>
        </FormField>
        <FormField name="renew_date" label="Renew date">
          <RenewDateCell
            value={token.renew_date}
            className="abm-renew-date-cell"
          />
        </FormField>
        <Dropdown
          searchable={false}
          options={options}
          onChange={(value: string) => {
            setSelectedTeamNames((prev) => ({ ...prev, macos_team: value }));
          }}
          value={selectedTeamNames.macos_team}
          label="macOS fleet"
          wrapperClassName={`${baseClass}__form-field form-field--macos`}
        />
        <Dropdown
          searchable={false}
          options={options}
          onChange={(value: string) => {
            setSelectedTeamNames((prev) => ({ ...prev, ios_team: value }));
          }}
          value={selectedTeamNames.ios_team}
          label="iOS fleet"
          wrapperClassName={`${baseClass}__form-field form-field--ios`}
        />
        <Dropdown
          searchable={false}
          options={options}
          onChange={(value: string) =>
            setSelectedTeamNames((prev) => ({ ...prev, ipados_team: value }))
          }
          value={selectedTeamNames.ipados_team}
          label="iPadOS fleet"
          wrapperClassName={`${baseClass}__form-field form-field--ipados`}
        />
        <Dropdown
          searchable={false}
          options={options}
          onChange={(value: string) =>
            setSelectedTeamNames((prev) => ({ ...prev, byod_team: value }))
          }
          value={selectedTeamNames.byod_team}
          label="BYOD fleet"
          wrapperClassName={`${baseClass}__form-field form-field--byod`}
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            className="save-abm-teams-loading"
            isLoading={isSaving}
          >
            Save
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditTeamsAbmModal;
