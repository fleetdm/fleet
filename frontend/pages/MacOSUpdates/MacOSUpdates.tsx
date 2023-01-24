import OperatingSystems from "pages/DashboardPage/cards/OperatingSystems";
import useInfoCard from "pages/DashboardPage/components/InfoCard";
import React from "react";
import OsMinVersionForm from "./components/OsMinVersionForm";
import NudgePreview from "./components/NudgePreview";

const baseClass = "mac-os-updates";

interface IMacOSUpdatesProps {
  teamId: number;
}

const MacOSUpdates = ({ teamId }: IMacOSUpdatesProps) => {
  const OperatingSystemCard = useInfoCard({
    title: "macOS versions",
    children: (
      <OperatingSystems
        currentTeamId={1}
        // TODO: uncomment when we integrate with page component
        // currentTeamId={teamId}
        selectedPlatform="darwin"
        showTitle
        includeName={false}
        setShowTitle={() => {
          return null;
        }}
      />
    ),
  });

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of macOS updates on hosts assigned
        to this team.
      </p>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__form-table-content`}>
          <div className={`${baseClass}__os-versions-card`}>
            {OperatingSystemCard}
          </div>
          <div className={`${baseClass}__os-version-form`}>
            <OsMinVersionForm />
          </div>
        </div>
        <div className={`${baseClass}__nudge-preview`}>
          <NudgePreview />
        </div>
      </div>
    </div>
  );
};

export default MacOSUpdates;
