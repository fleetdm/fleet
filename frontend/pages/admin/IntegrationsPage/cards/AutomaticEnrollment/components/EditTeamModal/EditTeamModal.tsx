import React, { useState, useContext, FormEvent, useCallback } from "react";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  APP_CONTEXT_NO_TEAM_ID,
  APP_CONTEX_NO_TEAM_SUMMARY,
} from "interfaces/team";
import configAPI from "services/entities/config";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IEditTeamModal {
  onCancel: () => void;
  defaultTeamName: string;
}

const baseClass = "edit-team-modal";

const EditTeamModal = ({
  onCancel,
  defaultTeamName,
}: IEditTeamModal): JSX.Element => {
  const { availableTeams, setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [selectedTeam, setSelectedTeam] = useState(defaultTeamName);

  const teamNameOptions = availableTeams
    ?.filter((t) => t.id >= APP_CONTEXT_NO_TEAM_ID)
    .map((teamSummary) => {
      return {
        value:
          teamSummary.name === APP_CONTEX_NO_TEAM_SUMMARY.name
            ? ""
            : teamSummary.name,
        label: teamSummary.name,
      };
    });

  const [isLoading, setIsLoading] = useState(false);

  const handleUpdateTeam = useCallback(
    async (newName: string) => {
      try {
        const configData = await configAPI.update({
          mdm: { apple_bm_default_team: newName },
        });
        renderFlash("success", "Default team updated successfully.");
        setConfig(configData);
      } catch (e) {
        renderFlash(
          "error",
          "Unable to update default team. Please try again."
        );
      } finally {
        onCancel();
      }
    },
    [renderFlash, setConfig, onCancel]
  );

  const onFormSubmit = useCallback(
    (evt: FormEvent<HTMLFormElement>) => {
      evt.preventDefault();
      setIsLoading(true);
      handleUpdateTeam(selectedTeam);
    },
    [selectedTeam, setIsLoading, handleUpdateTeam]
  );

  return (
    <Modal title="Edit team" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <div className="bottom-label">
          <Dropdown
            placeholder={selectedTeam}
            options={teamNameOptions}
            onChange={setSelectedTeam}
            value={selectedTeam}
            label="Team"
            helpText="macOS hosts will be added to this team when they're first unboxed."
          />
        </div>
        <div className="modal-cta-wrap">
          <Button type="submit" variant="brand" isLoading={isLoading}>
            Save
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EditTeamModal;
