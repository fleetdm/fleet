/**
 * Component when there is no hosts set up in fleet
 */
import React, { useState, useCallback } from "react";
import { useSelector } from "react-redux";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
import GenerateInstallerModal from "./GenerateInstallerModal";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

interface INoHostsProps {
  selectedTeam: number;
  teams?: ITeam[];
}

interface IRootState {
  app: {
    enrollSecret: IEnrollSecret[];
  };
}
const baseClass = "no-hosts";

const NoHosts = ({ selectedTeam, teams }: INoHostsProps): JSX.Element => {
  const globalSecret = useSelector(
    (state: IRootState) => state.app.enrollSecret
  );
  const [showGenerateInstallerModal, setShowGenerateInstallerModal] = useState(
    false
  );

  const toggleGenerateInstallerModal = useCallback(() => {
    setShowGenerateInstallerModal(!showGenerateInstallerModal);
  }, [showGenerateInstallerModal, setShowGenerateInstallerModal]);

  // TODO: Better way to make sure that team does not return as undefined
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }
    if (selectedTeam === 0) {
      return { name: "No team", secrets: globalSecret };
    }
    if (teams) {
      const team = teams.find((team) => team.id === selectedTeam);
      if (team) {
        return team;
      }
    }
    return { name: "No team", secrets: globalSecret };
  };

  const team = renderTeam();

  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <img src={RoboDogImage} alt="No Hosts" />
        <div>
          <h2>Add your devices to Fleet</h2>
          <p>Generate an installer to add your own devices.</p>
          <div className={`${baseClass}__no-hosts-button`}>
            <Button
              onClick={toggleGenerateInstallerModal}
              type="button"
              className="button button--brand"
            >
              Generate installer
            </Button>
          </div>
        </div>
      </div>
      {showGenerateInstallerModal ? (
        <GenerateInstallerModal
          onCancel={toggleGenerateInstallerModal}
          selectedTeam={team}
        />
      ) : null}
    </div>
  );
};

export default NoHosts;
