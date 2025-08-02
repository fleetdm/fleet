import React from "react";

import SectionHeader from "components/SectionHeader";

import { IScriptsCommonProps } from "../ScriptsNavItems";

const baseClass = "script-batch-progress";

export type IScriptBatchProgressProps = IScriptsCommonProps;

const ScriptBatchProgress = ({ router, teamId }: IScriptBatchProgressProps) => {
  return (
    <div className={baseClass}>
      <SectionHeader title="Batch progress" alignLeftHeaderVertically />
      <p>This is a placeholder for the Script Batch Progress component.</p>
    </div>
  );
};

export default ScriptBatchProgress;
