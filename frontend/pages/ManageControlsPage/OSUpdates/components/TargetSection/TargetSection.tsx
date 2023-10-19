import React from "react";

import SectionHeader from "components/SectionHeader";

import OsMinVersionForm from "../OsMinVersionForm";

const baseClass = "os-updates-target-section";

interface ITargetSectionProps {
  currentTeamId: number;
}

const TargetSection = ({ currentTeamId }: ITargetSectionProps) => {
  return (
    <div className={baseClass}>
      {/* <SectionHeader title="Target" /> */}
      <OsMinVersionForm currentTeamId={currentTeamId} key={currentTeamId} />
    </div>
  );
};

export default TargetSection;
