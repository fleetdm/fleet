import React from "react";

import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import EnrollSecretTable from "../EnrollSecretTable";

interface IEnrollSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  toggleDeleteSecretModal: () => void;
  setSelectedSecret: React.Dispatch<
    React.SetStateAction<IEnrollSecret | undefined>
  >;
  globalSecrets?: IEnrollSecret[] | undefined;
}

const baseClass = "enroll-secret-modal";

const EnrollSecretModal = ({
  onReturnToApp,
  selectedTeam,
  teams,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
  globalSecrets,
}: IEnrollSecretModal): JSX.Element => {
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam <= 0) {
      return { name: "No team", secrets: globalSecrets }; // TODO: Should "No team" be "Fleet" for free tier?
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  const addNewSecretClick = () => {
    setSelectedSecret(undefined);
    toggleSecretEditorModal();
  };
  const team = renderTeam();
  return (
    <Modal
      onExit={onReturnToApp}
      onEnter={onReturnToApp}
      title="Manage enroll secrets"
      className={baseClass}
    >
      <div className={`${baseClass} form`}>
        {team?.secrets?.length ? (
          <>
            <div className={`${baseClass}__description`}>
              Use these secret(s) to enroll hosts to <b>{renderTeam()?.name}</b>
              :
            </div>
            <EnrollSecretTable
              secrets={team?.secrets}
              toggleSecretEditorModal={toggleSecretEditorModal}
              toggleDeleteSecretModal={toggleDeleteSecretModal}
              setSelectedSecret={setSelectedSecret}
            />
          </>
        ) : (
          <>
            <div className={`${baseClass}__description`}>
              <p>
                <b>You have no enroll secrets.</b>
              </p>
              <p>
                Add secret(s) to enroll hosts to <b>{renderTeam()?.name}</b>.
              </p>
            </div>
          </>
        )}
        <div className={`${baseClass}__add-secret`}>
          <Button
            onClick={addNewSecretClick}
            className={`${baseClass}__add-secret-btn`}
            variant="text-icon"
          >
            <>
              Add secret <Icon name="plus" />
            </>
          </Button>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onReturnToApp} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default EnrollSecretModal;
