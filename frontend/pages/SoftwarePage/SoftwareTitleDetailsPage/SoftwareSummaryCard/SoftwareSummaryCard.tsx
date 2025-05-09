/** software/titles/:id > First section */

import React from "react";

import { InjectedRouter } from "react-router";

import {
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitleDetails,
} from "interfaces/software";

import Card from "components/Card";
import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary";
import TitleVersionsTable from "./TitleVersionsTable";

interface ISoftwareSummaryCard {
  title: ISoftwareTitleDetails;
  softwareId: number;
  teamId?: number;
  isAvailableForInstall?: boolean;
  isLoading?: boolean;
  router: InjectedRouter;
}

const baseClass = "software-summary-card";

const SoftwareSummaryCard = ({
  teamId,
  softwareId,
  isAvailableForInstall,
  title,
  isLoading = false,
  router,
}: ISoftwareSummaryCard) => {
  // Hide versions card for tgz_packages only
  if (title.source === "tgz_packages") return null;

  return (
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
        iconUrl={title.app_store_app ? title.app_store_app.icon_url : undefined}
      />
      <TitleVersionsTable
        router={router}
        data={title.versions ?? []}
        isLoading={isLoading}
        teamIdForApi={teamId}
        isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(title.source)}
        isAvailableForInstall={isAvailableForInstall}
        countsUpdatedAt={title.counts_updated_at}
      />
    </Card>
  );
};

export default SoftwareSummaryCard;
