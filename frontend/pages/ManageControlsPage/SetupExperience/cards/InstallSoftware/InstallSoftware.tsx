import React, { useState } from "react";

import SectionHeader from "components/SectionHeader";

import InstallSoftwarePreview from "./components/InstallSoftwarePreview";
import AddInstallSoftware from "./components/AddInstallSoftware";

const baseClass = "install-software";

interface IInstallSoftwareProps {}

const InstallSoftware = ({}: IInstallSoftwareProps) => {
  const [showSelectSoftwareModal, setShowSelectSoftwareModal] = useState(false);
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState([]);

  return (
    <div className={baseClass}>
      <SectionHeader title="Install software" />
      <div className={`${baseClass}__content`}>
        <AddInstallSoftware
          selectedSoftwareIds={selectedSoftwareIds}
          onAddSoftware={() => setShowSelectSoftwareModal(true)}
        />
        <InstallSoftwarePreview />
      </div>
    </div>
  );
};

export default InstallSoftware;
