import React from "react";

import SectionHeader from "components/SectionHeader";

const baseClass = "install-software";

interface IInstallSoftwareProps {}

const InstallSoftware = ({}: IInstallSoftwareProps) => {
  return (
    <div className={baseClass}>
      <SectionHeader title="Bootstrap package" />
      <div className={`${baseClass}__content`}>
        <div>col 1</div>
        <div>col 2</div>
      </div>
    </div>
  );
};

export default InstallSoftware;
