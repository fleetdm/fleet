import React from "react";

import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import EmptyState from "components/EmptyState";
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
      ? { name: "Unassigned", secrets: globalSecrets }
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
      {teamInfo?.secrets?.length ? (
        <div className={`${baseClass} form`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__description`}>
              Use these secret(s) to enroll hosts
              {primoMode || teamInfo?.name === "Unassigned" ? (
                ""
              ) : (
                <>
                  {" "}
                  to <b>{teamInfo?.name}</b>
                </>
              )}
              .
            </div>
            <div className={`${baseClass}__add-secret`}>
              <GitOpsModeTooltipWrapper
                entityType="secrets"
                position="right"
                tipOffset={8}
                renderChildren={(disableChildren) => (
                  <Button
                    disabled={disableChildren}
                    onClick={addNewSecretClick}
                    className={`${baseClass}__add-secret-btn`}
                    variant="brand-inverse-icon"
                    iconStroke
                  >
                    Add secret <Icon name="plus" color="core-fleet-green" />
                  </Button>
                )}
              />
            </div>
          </div>
          <EnrollSecretTable
            secrets={teamInfo?.secrets}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            setSelectedSecret={setSelectedSecret}
          />
        </div>
      ) : (
        <EmptyState
          variant="list"
          header="You have no enroll secrets"
          info={
            <>
              Add secret(s) to enroll hosts
              {primoMode || teamInfo?.name === "Unassigned" ? (
                ""
              ) : (
                <>
                  {" "}
                  to <b>{teamInfo?.name}</b>
                </>
              )}
              .
            </>
          }
          primaryButton={
            <GitOpsModeTooltipWrapper
              entityType="secrets"
              position="right"
              tipOffset={8}
              renderChildren={(disableChildren) => (
                <Button
                  disabled={disableChildren}
                  onClick={addNewSecretClick}
                  className={`${baseClass}__add-secret-btn`}
                  variant="brand-inverse-icon"
                  iconStroke
                >
                  Add secret <Icon name="plus" color="core-fleet-green" />
                </Button>
              )}
            />
          }
        />
      )}
      <div className="modal-cta-wrap">
        <Button onClick={onReturnToApp}>Close</Button>
      </div>
    </Modal>
  );
};

export default EnrollSecretModal;
