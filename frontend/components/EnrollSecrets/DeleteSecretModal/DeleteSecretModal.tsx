import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";
import CustomLink from "components/CustomLink";

interface IDeleteSecretModal {
  selectedTeam: number;
  teams: ITeam[];
  onDeleteSecret: () => void;
  toggleDeleteSecretModal: () => void;
  isUpdatingSecret: boolean;
}

const baseClass = "delete-secret-modal";

const DeleteSecretModal = ({
  selectedTeam,
  teams,
  onDeleteSecret,
  toggleDeleteSecretModal,
  isUpdatingSecret,
}: IDeleteSecretModal): JSX.Element => {
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team" };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  return (
    <Modal
      onExit={toggleDeleteSecretModal}
      onEnter={onDeleteSecret}
      title="Delete secret"
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          <p>
            This action will delete the secret used to enroll hosts to{" "}
            <b>{renderTeam()?.name}</b>.
          </p>
          <p>
            Any hosts that attempt to enroll to Fleet using this secret will be
            unable to enroll.
          </p>
          <p>
            Any enrolled hosts using this secret will not receive updates
            through Orbit including updates to agent options and command line
            flags.
          </p>
          <p>
            Follow this guide to{" "}
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/configuration-files#rotate-enroll-secrets"
              text="rotate enroll secrets"
              newTab
            />
            .
          </p>
          <p>You cannot undo this action.</p>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDeleteSecret}
            className="delete-loading"
            isLoading={isUpdatingSecret}
          >
            Delete
          </Button>
          <Button onClick={toggleDeleteSecretModal} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteSecretModal;
