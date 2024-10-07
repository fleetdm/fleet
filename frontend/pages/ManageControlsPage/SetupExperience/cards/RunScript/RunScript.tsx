import React from "react";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import AddRunScript from "./components/AddRunScript";
import RunScriptPreview from "./components/RunScriptPreview";
import CustomLink from "components/CustomLink";
import RunScriptUploader from "./components/RunScriptUploader";
import RunScriptCard from "./components/RunScriptCard";

const baseClass = "run-script";

interface IRunScriptProps {}

const RunScript = ({}: IRunScriptProps) => {
  const isLoading = false;
  const isError = false;

  const onUpload = () => {
    // refetchEnrollmentProfile();
  };

  const noPackageUploaded = true;

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
          <CustomLink newTab url="" text="Learn how" />
          {noPackageUploaded ? (
            <RunScriptUploader
              className={`${baseClass}__file-uploader`}
              onUpload={onUpload}
            />
          ) : (
            <RunScriptCard />
          )}
        </div>
        {/* currentTeamId={currentTeamId}
          softwareTitles={softwareTitles}
          onAddSoftware={() => setShowSelectSoftwareModal(true)} */}
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
