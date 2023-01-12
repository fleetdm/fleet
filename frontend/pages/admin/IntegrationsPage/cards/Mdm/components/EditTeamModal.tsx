import React, { useState, useContext, FormEvent } from "react";

import { AppContext } from "context/app";
import configAPI from "services/entities/config";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";

interface IEditTeamModal {
  onCancel: () => void;
  currentDefaultTeamName: string | undefined;
}

const baseClass = "edit-team-modal";

const EditTeamModal = ({
  onCancel,
  currentDefaultTeamName,
}: IEditTeamModal): JSX.Element => {
  const { availableTeams } = useContext(AppContext);

  const teamNameOptions = availableTeams?.map((teamSummary) => {
    return { value: teamSummary.name, label: teamSummary.name };
  });

  const [defaultTeamName, setDefaultTeamName] = useState(
    currentDefaultTeamName
  );

  const [isLoading, setIsLoading] = useState(false);

  const onFormSubmit = async (event: FormEvent) => {
    event.preventDefault();
    try {
      setIsLoading(true);
      await configAPI.update({
        mdm: { apple_bm_default_team: defaultTeamName },
      });
      setIsLoading(false);
    } finally {
      onCancel();
    }
  };

  return (
    <Modal title="Edit team" onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
        <div className="bottom-label">
          <Dropdown
            placeholder={defaultTeamName ?? "No team"}
            options={teamNameOptions}
            onChange={setDefaultTeamName}
            value={defaultTeamName ?? ""}
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
