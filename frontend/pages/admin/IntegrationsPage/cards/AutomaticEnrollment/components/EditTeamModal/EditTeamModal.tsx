import React, { useState, useContext, FormEvent } from "react";

import { AppContext } from "context/app";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import configAPI from "services/entities/config";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IEditTeamModal {
  onCancel: () => void;
  defaultTeamName: string;
  onUpdateSuccess: (newName: string) => void;
}

const baseClass = "edit-team-modal";

const EditTeamModal = ({
  onCancel,
  defaultTeamName,
  onUpdateSuccess,
}: IEditTeamModal): JSX.Element => {
  const { availableTeams } = useContext(AppContext);

  const [selectedTeam, setSelectedTeam] = useState(defaultTeamName);

  // TODO: Should this include "No team" as an option?
  const teamNameOptions = availableTeams
    ?.filter((t) => t.id > APP_CONTEXT_NO_TEAM_ID)
    .map((teamSummary) => {
      return { value: teamSummary.name, label: teamSummary.name };
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
    <Modal title="Edit team" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <div className="bottom-label">
          <Dropdown
            placeholder={selectedTeam}
            options={teamNameOptions}
            onChange={setSelectedTeam}
            value={selectedTeam}
            label="Team"
          />
          <p>
            macOS hosts will be added to this team when they&apos;re first
            unboxed.
          </p>
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
