/**
 * Component when there is no hosts set up in fleet
 */
import React from "react";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

interface INoHostsProps {
  setShowAddHostModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const baseClass = "no-hosts";

const NoHosts = ({ setShowAddHostModal }: INoHostsProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <img src={RoboDogImage} alt="No Hosts" />
        <div>
          <h2>Add your hosts to Fleet</h2>
          <p>Add your laptops and servers to securely monitor them.</p>
          <div className={`${baseClass}__no-hosts-button`}>
            <Button
              onClick={() => setShowAddHostModal(true)}
              type="button"
              className="button button--brand"
            >
              Add new hosts
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default NoHosts;
