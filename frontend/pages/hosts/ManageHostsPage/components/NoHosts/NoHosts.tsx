/**
 * Component when there is no hosts set up in fleet
 */
import React from "react";
import Button from "components/buttons/Button";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

interface INoHostsProps {
  toggleAddHostsModal: () => void;
  canEnrollHosts?: boolean;
  includesNameCardFilter?: boolean;
}

const baseClass = "no-hosts";

const NoHosts = ({
  toggleAddHostsModal,
  canEnrollHosts,
  includesNameCardFilter,
}: INoHostsProps): JSX.Element => {
  const renderContent = () => {
    if (includesNameCardFilter) {
      return (
        <div>
          <h1>No hosts match the current criteria</h1>
          <p>
            Expecting to see new hosts? Try again in a few seconds as the system
            catches up.
          </p>
        </div>
      );
    }

    if (canEnrollHosts) {
      return (
        <div>
          <h2>Add your devices to Fleet</h2>
          <p>Generate an installer to add your own devices.</p>
          <div className={`${baseClass}__no-hosts-button`}>
            <Button variant="brand" onClick={toggleAddHostsModal} type="button">
              Add hosts
            </Button>
          </div>
        </div>
      );
    }

    return (
      <div>
        <h2>Devices will show up here once they’re added to Fleet.</h2>
        <p>
          Expecting to see devices? Try again in a few seconds as the system
          catches up.
        </p>
      </div>
    );
  };

  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        {!includesNameCardFilter && <img src={RoboDogImage} alt="No Hosts" />}
        {renderContent()}
      </div>
    </div>
  );
};

export default NoHosts;
