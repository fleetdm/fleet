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

  const {
    data: softwareTitles,
    isLoading,
    isError,
    refetch: refetchSoftwareTitles,
  } = useQuery<ISoftwareTitlesResponse, AxiosError, ISoftwareTitle[]>(
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
      // onSuccess: (data) => {
      //   setSelectedSoftwareIds(
      //     data.reduce<number[]>((acc, software) => {
      //       if (software.install_during_setup) {
      //         acc.push(software.id);
      //       }
      //       return acc;
      //     }, [])
      //   );
      // },
    }
  );

  const onSave = () => {
    console.log("save");
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (softwareTitles) {
      return (
        <div className={`${baseClass}__content`}>
          <AddInstallSoftware
            softwareTitles={softwareTitles}
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
      {showSelectSoftwareModal && softwareTitles && (
        <SelectSoftwareModal
          softwareTitles={softwareTitles}
          onSave={onSave}
          onExit={() => setShowSelectSoftwareModal(false)}
        />
      )}
    </div>
  );
};

export default InstallSoftware;
