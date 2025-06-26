import React from "react";

import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import EnrollSecretTable from "../EnrollSecretTable";

interface IEnrollSecretModal {
  selectedTeamId: number;
  primoMode: boolean;
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
  selectedTeamId,
  primoMode,
  teams,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
  globalSecrets,
}: IEnrollSecretModal): JSX.Element => {
  const teamInfo =
    selectedTeamId <= 0
      ? { name: "No team", secrets: globalSecrets }
      : teams.find((team) => team.id === selectedTeamId);

  const addNewSecretClick = () => {
    setSelectedSecret(undefined);
    toggleSecretEditorModal();
  };
  return (
    <Modal
      onExit={onReturnToApp}
      onEnter={onReturnToApp}
      title="Manage enroll secrets"
      className={baseClass}
    >
      <div className={`${baseClass} form`}>
        {teamInfo?.secrets?.length ? (
          <>
            <div className={`${baseClass}__description`}>
              Use these secret(s) to enroll hosts
              {primoMode ? (
                ""
              ) : (
                <>
                  {" "}
                  to <b>{teamInfo?.name}</b>
                </>
              )}
              :
            </div>
            <EnrollSecretTable
              secrets={teamInfo?.secrets}
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
                Add secret(s) to enroll hosts
                {primoMode ? (
                  ""
                ) : (
                  <>
                    {" "}
                    to <b>{teamInfo?.name}</b>
                  </>
                )}
                .
              </p>
            </div>
          </>
        )}
        <div className={`${baseClass}__add-secret`}>
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                disabled={disableChildren}
                onClick={addNewSecretClick}
                className={`${baseClass}__add-secret-btn`}
                variant="text-icon"
                iconStroke
              >
                Add secret <Icon name="plus" />
              </Button>
            )}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onReturnToApp}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default EnrollSecretModal;
