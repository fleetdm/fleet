import React, { useCallback, useState } from "react";
import { Link } from "react-router";
import PATHS from "router/paths";
import permissionUtils from "utilities/permissions";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import EnrollSecretTable from "components/config/EnrollSecretTable";
import { ITeam } from "interfaces/team";
import { IConfig } from "interfaces/config";
import { IUser } from "interfaces/user";

interface IEnrollSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  isBasicTier: boolean;
  teams: ITeam[];
}

const baseClass = "enroll-secret-modal";

const EnrollSecretModal = ({
  onReturnToApp,
  selectedTeam,
  isBasicTier,
  teams,
}: IEnrollSecretModal): JSX.Element => {
  // const getSelectedEnrollSecrets = () => {
  //   if (selectedTeam === 0) {
  //     return this.state.globalSecrets;
  //   }
  //   return (
  //     this.teamSecrets.find((e) => e.id === selectedTeam.id)?.secrets || ""
  //   );
  // };

  const team = () => {
    if (selectedTeam === 0) {
      return { name: "No team", secrets: [{ secret: "globalsecretshere" }] };
    }
    return teams.find((team) => team.id == selectedTeam);
  };

  // const getSelectedEnrollSecrets = () => {
  //   if (selectedTeam === 0) {
  //     // return this.state.globalSecrets;
  //     return null;
  //   }
  //   return (
  //     this.teamSecrets.find((e) => e.id === selectedTeam.id)?.secrets || ""
  //   );
  // };

  return (
    <Modal onExit={onReturnToApp} title={"Enroll secret"} className={baseClass}>
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          Use these secret(s) to enroll devices to <b>{team()?.name}</b>:
        </div>
        <div className={`${baseClass}__secret-wrapper`}>
          {/* {isBasicTier && selectedTeam && (
            <EnrollSecretTable secrets={getSelectedEnrollSecrets()} />
          )} */}
          {/* {!isBasicTier && <EnrollSecretTable secrets={team()?.secrets.secret} />} */}
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
