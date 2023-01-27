import OperatingSystems from "pages/DashboardPage/cards/OperatingSystems";
import useInfoCard from "pages/DashboardPage/components/InfoCard";
import React from "react";
import OsMinVersionForm from "./components/OsMinVersionForm";
import NudgePreview from "./components/NudgePreview";

const baseClass = "mac-os-updates";

interface IMacOSUpdatesProps {
  location: {
    query: { team_id?: string };
  };
}

const MacOSUpdates = ({ location }: IMacOSUpdatesProps) => {
  const { team_id } = location.query;
  const teamId = team_id === undefined ? team_id : Number(team_id);

  const OperatingSystemCard = useInfoCard({
    title: "macOS versions",
    children: (
      <OperatingSystems
        currentTeamId={teamId}
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
            <OsMinVersionForm currentTeamId={teamId} key={teamId} />
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
