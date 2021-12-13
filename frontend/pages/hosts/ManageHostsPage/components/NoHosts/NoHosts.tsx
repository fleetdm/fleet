/**
 * Component when there is no hosts set up in fleet
 */
import React from "react";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

interface INoHostsProps {
  toggleGenerateInstallerModal: () => void;
  canEnrollHosts?: boolean;
}

const baseClass = "no-hosts";

const NoHosts = ({
  toggleGenerateInstallerModal,
  canEnrollHosts,
}: INoHostsProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <img src={RoboDogImage} alt="No Hosts" />
        {canEnrollHosts ? (
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
        ) : (
          <div>
            <h2>Devices will show up here once they’re added to Fleet.</h2>
            <p>
              Expecting to see devices? Try again in a few seconds as the system
              catches up.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default NoHosts;
