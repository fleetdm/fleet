/** software/titles/:id > First section */

import React, { useState } from "react";

import { InjectedRouter } from "react-router";

import {
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitleDetails,
  isSoftwarePackage,
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
  softwareInstaller?: any;
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
  // Hide versions table for tgz_packages only
  const showVersionsTable = title.source !== "tgz_packages";
  const [showEditIconModal, setShowEditIconModal] = useState(false);

  const onClickEditIcon = () => {
    setShowEditIconModal(!showEditIconModal);
  };

  return (
    <>
      <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
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
          iconUrl={
            title.app_store_app ? title.app_store_app.icon_url : undefined
          }
          onClickEditIcon={softwareInstaller ? onClickEditIcon : undefined}
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
      {showEditIconModal && teamId && (
        <EditIconModal
          softwareId={softwareId}
          teamIdForApi={teamId}
          software={softwareInstaller}
          onExit={() => setShowEditIconModal(false)}
          refetchSoftwareTitle={refetchSoftwareTitle}
          installerType={
            isSoftwarePackage(softwareInstaller) ? "package" : "vpp"
          }
          previewInfo={{
            name: title.name,
            type: formatSoftwareType(title),
            versions: title.versions?.length ?? 0,
            countsUpdatedAt: title.counts_updated_at,
          }}
        />
      )}
    </>
  );
};

export default SoftwareSummaryCard;
