import React, { useState } from "react";
import { useQuery } from "react-query";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import mdmAPI from "services/entities/mdm";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import CustomLink from "components/CustomLink";

import RunScriptPreview from "./components/RunScriptPreview";
import RunScriptUploader from "./components/RunScriptUploader";
import RunScriptCard from "./components/RunScriptCard";

const baseClass = "run-script";

interface IRunScriptProps {
  currentTeamId: number;
}

const RunScript = ({ currentTeamId }: IRunScriptProps) => {
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const { data: script, isLoading, isError } = useQuery(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const onUpload = () => {
    // refetchEnrollmentProfile();
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
            <RunScriptUploader
              currentTeamId={currentTeamId}
              onUpload={onUpload}
            />
          ) : (
            <RunScriptCard
              script={script}
              onDelete={() => setShowDeleteScriptModal(true)}
            />
          )}
        </div>
        <RunScriptPreview />
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
    </div>
  );
};

export default RunScript;
