/** software/titles/:id > First section */

import React, { useContext, useState } from "react";
import { AppContext } from "context/app";

import { InjectedRouter } from "react-router";

import {
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitleDetails,
  isSoftwarePackage,
  ISoftwarePackage,
  IAppStoreApp,
  NO_VERSION_OR_HOST_DATA_AVAIL_SOURCES,
} from "interfaces/software";

import Card from "components/Card";
import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary";
import TitleVersionsTable from "./TitleVersionsTable";
import EditIconModal from "../EditIconModal";

interface ISoftwareSummaryCard {
  title: ISoftwareTitleDetails;
  softwareId: number;
  teamId?: number;
  isAvailableForInstall?: boolean;
  isLoading?: boolean;
  router: InjectedRouter;
  refetchSoftwareTitle: () => void;
  softwareInstaller?: ISoftwarePackage | IAppStoreApp;
}

const baseClass = "software-summary-card";

const SoftwareSummaryCard = ({
  teamId,
  softwareId,
  isAvailableForInstall,
  title,
  isLoading = false,
  router,
  softwareInstaller,
  refetchSoftwareTitle,
}: ISoftwareSummaryCard) => {
  const { source } = title;

  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);

  const [iconUploadedAt, setIconUploadedAt] = useState("");

  // Hide versions table for tgz_packages, sh_packages, & ps1_packages only
  const showVersionsTable = !NO_VERSION_OR_HOST_DATA_AVAIL_SOURCES.includes(
    source
  );

  const hasEditPermissions =
    isGlobalAdmin || isGlobalMaintainer || isTeamMaintainerOrTeamAdmin;
  const canEditIcon =
    softwareInstaller &&
    typeof teamId === "number" &&
    teamId >= 0 &&
    hasEditPermissions;

  const [showEditIconModal, setShowEditIconModal] = useState(false);

  const onClickEditIcon = () => {
    setShowEditIconModal(!showEditIconModal);
  };

  return (
    <>
      <Card borderRadiusSize="xxlarge" className={baseClass}>
        <SoftwareDetailsSummary
          title={title.name}
          type={formatSoftwareType(title)}
          versions={title.versions?.length ?? 0}
          hosts={title.hosts_count}
          countsUpdatedAt={title.counts_updated_at}
          queryParams={{
            software_title_id: softwareId,
            team_id: teamId,
          }}
          name={title.name}
          source={title.source}
          iconUrl={title.icon_url}
          iconUploadedAt={iconUploadedAt}
          onClickEditIcon={canEditIcon ? onClickEditIcon : undefined}
        />
        {showVersionsTable && (
          <TitleVersionsTable
            router={router}
            data={title.versions ?? []}
            isLoading={isLoading}
            teamIdForApi={teamId}
            isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(title.source)}
            isAvailableForInstall={isAvailableForInstall}
            countsUpdatedAt={title.counts_updated_at}
          />
        )}
      </Card>
      {showEditIconModal &&
        typeof teamId === "number" &&
        teamId >= 0 &&
        softwareInstaller && (
          <EditIconModal
            softwareId={softwareId}
            teamIdForApi={teamId}
            software={softwareInstaller}
            onExit={() => setShowEditIconModal(false)}
            refetchSoftwareTitle={refetchSoftwareTitle}
            iconUploadedAt={iconUploadedAt}
            setIconUploadedAt={setIconUploadedAt}
            installerType={
              isSoftwarePackage(softwareInstaller) ? "package" : "vpp"
            }
            previewInfo={{
              name: title.name,
              type: formatSoftwareType(title),
              source: title.source,
              currentIconUrl: title.icon_url,
              versions: title.versions?.length ?? 0,
              countsUpdatedAt: title.counts_updated_at,
            }}
          />
        )}
    </>
  );
};

export default SoftwareSummaryCard;
