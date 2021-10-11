/**
 * Component when there is no hosts set up in fleet
 */
import React, { useState, useCallback } from "react";

import Button from "components/buttons/Button";
import GenerateInstallerModal from "./GenerateInstallerModal";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

const baseClass = "no-hosts";

const NoHosts = (): JSX.Element => {
  const [showGenerateInstallerModal, setShowGenerateInstallerModal] = useState(
    false
  );

  const toggleGenerateInstallerModal = useCallback(() => {
    setShowGenerateInstallerModal(!showGenerateInstallerModal);
  }, [showGenerateInstallerModal, setShowGenerateInstallerModal]);

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
        <GenerateInstallerModal onCancel={toggleGenerateInstallerModal} />
      ) : null}
    </div>
  );
};

export default NoHosts;
