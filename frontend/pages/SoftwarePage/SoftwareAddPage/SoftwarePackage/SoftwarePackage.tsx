import React from "react";

const baseClass = "software-package";

interface ISoftwarePackageProps {}

const SoftwarePackage = ({}: ISoftwarePackageProps) => {
  return <div className={baseClass}>Sofware package page</div>;
};

export default SoftwarePackage;
