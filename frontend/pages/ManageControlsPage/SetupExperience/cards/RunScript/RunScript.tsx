import React from "react";

import SectionHeader from "components/SectionHeader";

const baseClass = "run-script";

interface IRunScriptProps {}

const RunScript = ({}: IRunScriptProps) => {
  const renderContent = () => {};

  return (
    <div className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
    </div>
  );
};

export default RunScript;
