import React from "react";
import { useSelector } from "react-redux";
import { useQuery } from "react-query";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import enrollSecretsAPI from "services/entities/enroll_secret";
// @ts-ignore
import EnrollSecretTable from "components/EnrollSecretTable";
import { ITeam } from "interfaces/team";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";

import PlusIcon from "../../../../../../assets/images/icon-plus-16x16@2x.png";

interface IEnrollSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  isPremiumTier: boolean;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  toggleDeleteSecretModal: () => void;
  setSelectedSecret: React.Dispatch<
    React.SetStateAction<IEnrollSecret | undefined>
  >;
}

const baseClass = "enroll-secret-modal";

const EnrollSecretModal = ({
  onReturnToApp,
  selectedTeam,
  isPremiumTier,
  teams,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
}: IEnrollSecretModal): JSX.Element => {
  const {
    isLoading: isGlobalSecretsLoading,
    data: globalSecrets,
    error: globalSecretsError,
    refetch: refetchGlobalSecrets,
  } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
    ["global secrets"],
    () => enrollSecretsAPI.getGlobalEnrollSecrets(),
    {
      select: (data: IEnrollSecretsResponse) => data.secrets,
    }
  );

  // TODO: Revisit this to make the API be called only if there is a currentTeam 11/11 RP SG
  // const {
  //   isLoading: isTeamSecretsLoading,
  //   data: teamSecrets,
  //   error: teamSecretsError,
  //   refetch: refetchTeamSecrets,
  // } = useQuery<IEnrollSecretsResponse, Error, IEnrollSecret[]>(
  //   ["team secrets", selectedTeam],
  //   () => {
  //     if (selectedTeam) {
  //       return enrollSecretsAPI.getTeamEnrollSecrets(selectedTeam);
  //     }
  //     return { secrets: [] };
  //   },
  //   {
  //     enabled: !!selectedTeam,
  //     select: (data: IEnrollSecretsResponse) => data.secrets,
  //   }
  // );

  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team", secrets: globalSecrets };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  const addNewSecretClick = () => {
    setSelectedSecret(undefined);
    toggleSecretEditorModal();
  };

  return (
    <Modal onExit={onReturnToApp} title={"Enroll secret"} className={baseClass}>
      <div className={baseClass}>
        {renderTeam()?.secrets?.length ? (
          <>
            <div className={`${baseClass}__description`}>
              Use these secret(s) to enroll hosts to <b>{renderTeam()?.name}</b>
              :
            </div>
            <div className={`${baseClass}__secret-wrapper`}>
              {isPremiumTier && (
                <EnrollSecretTable
                  secrets={renderTeam()?.secrets}
                  toggleSecretEditorModal={toggleSecretEditorModal}
                  toggleDeleteSecretModal={toggleDeleteSecretModal}
                  setSelectedSecret={setSelectedSecret}
                />
              )}
              {!isPremiumTier && (
                <EnrollSecretTable
                  secrets={renderTeam()?.secrets}
                  toggleSecretEditorModal={toggleSecretEditorModal}
                  toggleDeleteSecretModal={toggleDeleteSecretModal}
                  setSelectedSecret={setSelectedSecret}
                />
              )}
            </div>
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
