import React from "react";

const baseClass = "hosts-enrolled-card";

const HostsEnrolledCard = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <h2 className={`${baseClass}__title`}>Hosts enrolled</h2>
      <div className={`${baseClass}__placeholder`}>
        Hosts enrolled chart coming soon
      </div>
    </div>
  );
};

export default HostsEnrolledCard;
