import React, { useState, useContext, FormEvent } from "react";

import { AppContext } from "context/app";
import {
  APP_CONTEXT_NO_TEAM_ID,
  APP_CONTEX_NO_TEAM_SUMMARY,
} from "interfaces/team";
import configAPI from "services/entities/config";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IRenameTeamModal {
  onCancel: () => void;
  defaultTeamName: string;
  onUpdateSuccess: (newName: string) => void;
}

const baseClass = "edit-team-modal";

const RenameTeamModal = ({
  onCancel,
  defaultTeamName,
  onUpdateSuccess,
}: IRenameTeamModal): JSX.Element => {
  const { availableTeams } = useContext(AppContext);

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

  const onFormSubmit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      setIsLoading(true);
      const configData = await configAPI.update({
        mdm: { apple_bm_default_team: selectedTeam },
      });
      setIsLoading(false);
      onUpdateSuccess(configData.mdm.apple_bm_default_team);
    } finally {
      onCancel();
    }
  };

  return (
    <Modal title="Rename team" onExit={onCancel} className={baseClass}>
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

export default RenameTeamModal;
