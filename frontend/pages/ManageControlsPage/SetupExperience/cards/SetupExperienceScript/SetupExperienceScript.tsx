import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import mdmAPI, {
  IGetSetupExperienceScriptResponse,
} from "services/entities/mdm";

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
    error: scriptError,
    isLoading,
    isError,
    refetch: refetchScript,
    remove: removeScriptFromCache,
  } = useQuery<IGetSetupExperienceScriptResponse, AxiosError>(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS, retry: false }
  );

  const onUpload = () => {
    refetchScript();
  };

  const onDelete = () => {
    removeScriptFromCache();
    setShowDeleteScriptModal(false);
    refetchScript();
  };

  const scriptUploaded = true;

  const renderContent = () => {
    if (isLoading) {
      <Spinner />;
    }

    if (isError && scriptError.status !== 404) {
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
            <>
              <p className={`${baseClass}__run-message`}>
                Script will run during setup:
              </p>
              <SetupExperienceScriptCard
                script={script}
                onDelete={() => setShowDeleteScriptModal(true)}
              />
            </>
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
      {showDeleteScriptModal && script && (
        <DeleteSetupExperienceScriptModal
          currentTeamId={currentTeamId}
          scriptName={script.name}
          onDeleted={onDelete}
          onExit={() => setShowDeleteScriptModal(false)}
        />
      )}
    </div>
  );
};

export default SetupExperienceScript;
