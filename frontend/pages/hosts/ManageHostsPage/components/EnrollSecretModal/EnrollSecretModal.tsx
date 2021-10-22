import React from "react";
import { useSelector } from "react-redux";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import EnrollSecretTable from "components/EnrollSecretTable";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import PlusIcon from "../../../../../../assets/images/icon-plus-16x16@2x.png";

interface IEnrollSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  isPremiumTier: boolean;
  teams: ITeam[];
  toggleAddSecretModal: () => void;
  toggleEditSecretModal: () => void;
  toggleDeleteSecretModal: () => void;
}

interface IRootState {
  app: {
    enrollSecret: IEnrollSecret[];
  };
}

const baseClass = "enroll-secret-modal";

const EnrollSecretModal = ({
  onReturnToApp,
  selectedTeam,
  isPremiumTier,
  teams,
  toggleAddSecretModal,
  toggleEditSecretModal,
  toggleDeleteSecretModal,
}: IEnrollSecretModal): JSX.Element => {
  const globalSecret = useSelector(
    (state: IRootState) => state.app.enrollSecret
  );

  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team", secrets: globalSecret };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  return (
    <Modal onExit={onReturnToApp} title={"Enroll secret"} className={baseClass}>
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          Use these secret(s) to enroll devices to <b>{renderTeam()?.name}</b>:
        </div>
        <div className={`${baseClass}__secret-wrapper`}>
          {isPremiumTier && (
            <EnrollSecretTable
              secrets={renderTeam()?.secrets}
              toggleEditSecretModal={toggleEditSecretModal}
              toggleDeleteSecretModal={toggleDeleteSecretModal}
            />
          )}
          {!isPremiumTier && (
            <EnrollSecretTable
              secrets={renderTeam()?.secrets}
              toggleEditSecretModal={toggleEditSecretModal}
              toggleDeleteSecretModal={toggleDeleteSecretModal}
            />
          )}
        </div>
        <div className={`${baseClass}__add-secret`}>
          <Button
            onClick={toggleAddSecretModal}
            className={`${baseClass}__add-secret-btn`}
            variant="text-icon"
          >
            <>
              Add secret <img src={PlusIcon} alt="Add secret icon" />
            </>
          </Button>
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onReturnToApp} className="button button--brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default EnrollSecretModal;
