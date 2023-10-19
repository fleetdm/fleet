import LastUpdatedText from "components/LastUpdatedText";
import SectionHeader from "components/SectionHeader";
import OperatingSystems from "pages/DashboardPage/cards/OperatingSystems";
import useInfoCard from "pages/DashboardPage/components/InfoCard";
import React from "react";

const baseClass = "os-updates-current-version-section";

interface ICurrentVersionSectionProps {
  currentTeamId: number;
}

const CurrentVersionSection = ({
  currentTeamId,
}: ICurrentVersionSectionProps) => {
  const OperatingSystemCard = useInfoCard({
    title: "macOS versions",
    children: (
      <OperatingSystems
        currentTeamId={currentTeamId}
        selectedPlatform="darwin"
        showTitle
        showDescription={false}
        includeNameColumn={false}
        setShowTitle={() => {
          return null;
        }}
      />
    ),
  });
  return (
    <div className={baseClass}>
      <SectionHeader
        title="Current versions"
        subTitle={
          <LastUpdatedText
            lastUpdatedAt={new Date().toISOString()}
            whatToRetrieve={"operating systems"}
          />
        }
      />
      {OperatingSystemCard}
    </div>
  );
};

export default CurrentVersionSection;
