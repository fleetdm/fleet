import React from "react";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  currentTeamId?: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  return <div className={baseClass}>bootstrap package</div>;
};

export default BootstrapPackage;
