/**
 * Component when there is no hosts set up in fleet
 */
import React from "react";

import RoboDogImage from "../../../../../../assets/images/robo-dog-176x144@2x.png";

const baseClass = "no-hosts";

const NoHosts = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <img src={RoboDogImage} alt="No Hosts" />
        <div>
          <h1>It&#39;s kinda empty in here...</h1>
          <h2>Get started adding hosts to Fleet.</h2>
          <p>Add your laptops and servers to securely monitor them.</p>
          <div className={`${baseClass}__no-hosts-contact`}>
            <p>Still having trouble?</p>
            <a href="https://github.com/fleetdm/fleet/issues">
              File a GitHub issue
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default NoHosts;
