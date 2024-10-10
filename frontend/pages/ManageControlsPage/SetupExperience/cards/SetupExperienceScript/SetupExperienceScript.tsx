import React, { useState } from "react";
import { useQuery } from "react-query";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import mdmAPI from "services/entities/mdm";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import CustomLink from "components/CustomLink";

import SetupExperiencePreview from "./components/SetupExperienceScriptPreview";
import SetupExperienceScriptUploader from "./components/SetupExperienceScriptUploader";
import SetupExperienceScriptCard from "./components/SetupExperienceScriptCard";
import DeleteSetupExperienceScriptModal from "./components/DeleteSetupExperienceScriptModal";

const baseClass = "setup-experience-script";

interface ISetupExperienceScriptProps {
  currentTeamId: number;
}

const SetupExperienceScript = ({
  currentTeamId,
}: ISetupExperienceScriptProps) => {
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const {
    data: script,
    isLoading,
    isError,
    refetch: refetchScript,
  } = useQuery(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const onUpload = () => {
    refetchScript();
  };

  const onDelete = () => {
    setShowDeleteScriptModal(false);
    refetchScript();
  };

  const scriptUploaded = true;

  const renderContent = () => {
    if (isLoading) {
      <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__description-container`}>
          <p className={`${baseClass}__description`}>
            Upload a script to run on hosts that automatically enroll to Fleet.
          </p>
          <CustomLink
            className={`${baseClass}__learn-how-link`}
            newTab
            url=""
            text="Learn how"
          />
          {!scriptUploaded || !script ? (
            <SetupExperienceScriptUploader
              currentTeamId={currentTeamId}
              onUpload={onUpload}
            />
          ) : (
            <SetupExperienceScriptCard
              script={script}
              onDelete={() => setShowDeleteScriptModal(true)}
            />
          )}
        </div>
        <SetupExperiencePreview />
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
      {showDeleteScriptModal && (
        <DeleteSetupExperienceScriptModal
          onDeleted={onDelete}
          onExit={() => setShowDeleteScriptModal(false)}
        />
      )}
    </div>
  );
};

export default SetupExperienceScript;
