import React from "react";
import { useSelector } from "react-redux";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IDeleteSecretModal {
  selectedTeam: number;
  teams: ITeam[];
  onDeleteSecret: () => void;
  toggleDeleteSecretModal: () => void;
}

interface IRootState {
  app: {
    enrollSecret: IEnrollSecret[];
  };
}

const baseClass = "delete-secret-modal";

const DeleteSecretModal = ({
  selectedTeam,
  teams,
  onDeleteSecret,
  toggleDeleteSecretModal,
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
      title={"Delete secret"}
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
          <p>You cannot undo this action.</p>
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={toggleDeleteSecretModal}
            variant="inverse-alert"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="alert"
            onClick={onDeleteSecret}
          >
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteSecretModal;
