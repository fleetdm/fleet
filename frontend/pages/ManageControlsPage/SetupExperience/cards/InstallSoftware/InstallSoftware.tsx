import React, { useState } from "react";
import { useQuery } from "react-query";

import softwareAPI from "services/entities/software";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import InstallSoftwarePreview from "./components/InstallSoftwarePreview";
import AddInstallSoftware from "./components/AddInstallSoftware";

const baseClass = "install-software";

interface IInstallSoftwareProps {
  currentTeamId: number;
}

const InstallSoftware = ({ currentTeamId }: IInstallSoftwareProps) => {
  const [showSelectSoftwareModal, setShowSelectSoftwareModal] = useState(false);
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState([]);

  const { data, isLoading, isError } = useQuery(
    ["install-software", currentTeamId],
    () =>
      softwareAPI.getSoftwareTitles({
        teamId: currentTeamId,
        availableForInstall: true,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
    }
  );

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (data) {
      return (
        <div className={`${baseClass}__content`}>
          <AddInstallSoftware
            selectedSoftwareIds={selectedSoftwareIds}
            onAddSoftware={() => setShowSelectSoftwareModal(true)}
          />
          <InstallSoftwarePreview />
        </div>
      );
    }

    return null;
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Install software" />
      <>{renderContent()}</>
    </div>
  );
};

export default InstallSoftware;
