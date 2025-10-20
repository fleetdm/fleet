import React from "react";

import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import Card from "components/Card";
import EmptyTable from "components/EmptyTable";
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
            <div className={`${baseClass}__header`}>
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
                .
              </div>
              <div className={`${baseClass}__add-secret`}>
                <GitOpsModeTooltipWrapper
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
          </>
        ) : (
          <Card color="grey" paddingSize="small">
            <EmptyTable
              header="You have no enroll secrets."
              info={
                <>
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
                </>
              }
              primaryButton={
                <GitOpsModeTooltipWrapper
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
          </Card>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onReturnToApp}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default EnrollSecretModal;
