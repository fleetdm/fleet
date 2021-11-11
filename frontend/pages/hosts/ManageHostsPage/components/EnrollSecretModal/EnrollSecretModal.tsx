import React from "react";
import { useSelector } from "react-redux";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import EnrollSecretTable from "components/EnrollSecretTable";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IEnrollSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  isPremiumTier: boolean;
  teams: ITeam[];
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
            <EnrollSecretTable secrets={renderTeam()?.secrets} />
          )}
          {!isPremiumTier && (
            <EnrollSecretTable secrets={renderTeam()?.secrets} />
          )}
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
