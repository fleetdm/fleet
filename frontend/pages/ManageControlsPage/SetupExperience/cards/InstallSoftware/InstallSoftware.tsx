import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { ISoftwareTitle } from "interfaces/software";
import softwareAPI, {
  ISoftwareTitlesResponse,
} from "services/entities/software";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import InstallSoftwarePreview from "./components/InstallSoftwarePreview";
import AddInstallSoftware from "./components/AddInstallSoftware";
import SelectSoftwareModal from "./components/SelectSoftwareModal";

const baseClass = "install-software";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

interface IInstallSoftwareProps {
  currentTeamId: number;
}

const InstallSoftware = ({ currentTeamId }: IInstallSoftwareProps) => {
  const [showSelectSoftwareModal, setShowSelectSoftwareModal] = useState(false);
  const [selectedSoftwareIds, setSelectedSoftwareIds] = useState<number[]>([]);

  const { isLoading, isError } = useQuery<
    ISoftwareTitlesResponse,
    AxiosError,
    ISoftwareTitle[]
  >(
    ["install-software", currentTeamId],
    () =>
      softwareAPI.getSoftwareTitles({
        teamId: currentTeamId,
        availableForInstall: true,
        perPage: PER_PAGE_SIZE,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
      onSuccess: (softwareTitles) => {
        setSelectedSoftwareIds(softwareTitles.map((software) => software.id));
      },
    }
  );

  const onSave = () => {};

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (selectedSoftwareIds) {
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
      {showSelectSoftwareModal && (
        <SelectSoftwareModal
          onSave={onSave}
          onExit={() => setShowSelectSoftwareModal(false)}
        />
      )}
    </div>
  );
};

export default InstallSoftware;
