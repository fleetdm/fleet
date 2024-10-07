import React from "react";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import AddRunScript from "./components/AddRunScript";
import RunScriptPreview from "./components/RunScriptPreview";

const baseClass = "run-script";

interface IRunScriptProps {}

const RunScript = ({}: IRunScriptProps) => {
  const isLoading = false;
  const isError = false;

  const renderContent = () => {
    if (isLoading) {
      <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <AddRunScript />
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
