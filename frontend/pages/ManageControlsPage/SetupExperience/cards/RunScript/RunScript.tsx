import React from "react";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import AddRunScript from "./components/AddRunScript";

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

    return <AddRunScript />;
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
    </div>
  );
};

export default RunScript;
