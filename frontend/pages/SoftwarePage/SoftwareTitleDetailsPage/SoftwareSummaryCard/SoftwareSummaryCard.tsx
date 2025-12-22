/** software/titles/:id > First section */

import React, { useState } from "react";

import { InjectedRouter } from "react-router";

import { useSoftwareInstaller } from "hooks/useSoftwareInstallerMeta";
import {
  formatSoftwareType,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitleDetails,
  NO_VERSION_OR_HOST_DATA_SOURCES,
  IAppStoreApp,
} from "interfaces/software";

import Card from "components/Card";
import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary";
import TitleVersionsTable from "./TitleVersionsTable";
import EditIconModal from "../EditIconModal";
import EditSoftwareModal from "../EditSoftwareModal";
import EditConfigurationModal from "../EditConfigurationModal";

interface ISoftwareSummaryCard {
  softwareTitle: ISoftwareTitleDetails;
  softwareId: number;
  teamId?: number;
  isAvailableForInstall?: boolean;
  isLoading?: boolean;
  router: InjectedRouter;
  refetchSoftwareTitle: () => void;
  onToggleViewYaml: () => void;
}

const baseClass = "software-summary-card";

const SoftwareSummaryCard = ({
  softwareTitle,
  softwareId,
  teamId,
  isAvailableForInstall,
  isLoading = false,
  router,
  refetchSoftwareTitle,
  onToggleViewYaml,
}: ISoftwareSummaryCard) => {
  const installerResult = useSoftwareInstaller(softwareTitle);

  const [iconUploadedAt, setIconUploadedAt] = useState("");
  const [showEditIconModal, setShowEditIconModal] = useState(false);
  const [showEditSoftwareModal, setShowEditSoftwareModal] = useState(false);
  const [showEditConfigurationModal, setShowEditConfigurationModal] = useState(
    false
  );
  const [
    showEditAutoUpdateConfigModal,
    setShowEditAutoUpdateConfigModal,
  ] = useState(false);

  // Hide versions table for tgz_packages, sh_packages, & ps1_packages and when no hosts have the
  // software installed
  const showVersionsTable =
    !!softwareTitle.hosts_count &&
    !NO_VERSION_OR_HOST_DATA_SOURCES.includes(softwareTitle.source);

  // If there is no installer (no package/app), bail out of installer‑related UI.
  if (!installerResult) {
    // when no installer, no edit actions:
    return (
      <>
        <Card borderRadiusSize="xxlarge" className={baseClass}>
          <SoftwareDetailsSummary
            displayName={softwareTitle.display_name || softwareTitle.name}
            type={formatSoftwareType(softwareTitle)}
            versions={softwareTitle.versions?.length ?? 0}
            hostCount={softwareTitle.hosts_count}
            countsUpdatedAt={softwareTitle.counts_updated_at}
            queryParams={{ software_title_id: softwareId, team_id: teamId }}
            name={softwareTitle.name}
            source={softwareTitle.source}
            iconUrl={softwareTitle.icon_url}
            iconUploadedAt={iconUploadedAt}
          />
          {showVersionsTable && (
            <TitleVersionsTable
              router={router}
              data={softwareTitle.versions ?? []}
              isLoading={isLoading}
              teamIdForApi={teamId}
              isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(
                softwareTitle.source
              )}
              isAvailableForInstall={isAvailableForInstall}
              countsUpdatedAt={softwareTitle.counts_updated_at}
            />
          )}
        </Card>
      </>
    );
  }

  const { meta } = installerResult;
  const {
    softwareInstaller,
    installerType,
    isIosOrIpadosApp,
    isAndroidPlayStoreApp,
    canManageSoftware,
  } = meta;

  const canEditAppearance = canManageSoftware;

  const canEditSoftware = canManageSoftware;

  const canEditConfiguration = canManageSoftware && isAndroidPlayStoreApp;

  const canEditAutoUpdateConfig =
    softwareTitle.app_store_app &&
    (softwareTitle.app_store_app.platform === "ios" ||
      softwareTitle.app_store_app.platform === "ipados") &&
    canManageSoftware;

  const onClickEditAppearance = () => setShowEditIconModal(true);
  const onClickEditSoftware = () => setShowEditSoftwareModal(true);
  const onClickEditConfiguration = () => setShowEditConfigurationModal(true);
  const onClickEditAutoUpdateConfig = () =>
    setShowEditAutoUpdateConfigModal(true);

  return (
    <>
      <Card borderRadiusSize="xxlarge" className={baseClass}>
        <SoftwareDetailsSummary
          displayName={softwareTitle.display_name || softwareTitle.name}
          type={formatSoftwareType(softwareTitle)}
          versions={softwareTitle.versions?.length ?? 0}
          hostCount={softwareTitle.hosts_count}
          countsUpdatedAt={softwareTitle.counts_updated_at}
          queryParams={{
            software_title_id: softwareId,
            team_id: teamId,
          }}
          name={softwareTitle.name}
          source={softwareTitle.source}
          iconUrl={softwareTitle.icon_url}
          iconUploadedAt={iconUploadedAt}
          canManageSoftware={canManageSoftware}
          onClickEditAppearance={
            canEditAppearance ? onClickEditAppearance : undefined
          }
          onClickEditSoftware={
            canEditSoftware ? onClickEditSoftware : undefined
          }
          onClickEditConfiguration={
            canEditConfiguration ? onClickEditConfiguration : undefined
          }
          onClickEditAutoUpdateConfig={
            canEditAutoUpdateConfig ? onClickEditAutoUpdateConfig : undefined
          }
        />
        {showVersionsTable && (
          <TitleVersionsTable
            router={router}
            data={softwareTitle.versions ?? []}
            isLoading={isLoading}
            teamIdForApi={teamId}
            isIPadOSOrIOSApp={isIosOrIpadosApp}
            isAvailableForInstall={isAvailableForInstall}
            countsUpdatedAt={softwareTitle.counts_updated_at}
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
            installerType={installerType}
            previewInfo={{
              name: softwareTitle.display_name || softwareTitle.name,
              titleName: softwareTitle.name,
              type: formatSoftwareType(softwareTitle),
              source: softwareTitle.source,
              currentIconUrl: softwareTitle.icon_url,
              versions: softwareTitle.versions?.length ?? 0,
              countsUpdatedAt: softwareTitle.counts_updated_at,
              selfServiceVersion: softwareInstaller.version,
            }}
          />
        )}
      {showEditSoftwareModal && softwareInstaller && teamId && (
        <EditSoftwareModal
          router={router}
          softwareId={softwareId}
          teamId={teamId}
          softwareInstaller={softwareInstaller}
          onExit={() => setShowEditSoftwareModal(false)}
          refetchSoftwareTitle={refetchSoftwareTitle}
          installerType={installerType}
          openViewYamlModal={onToggleViewYaml}
          isIosOrIpadosApp={isIosOrIpadosApp}
          name={softwareTitle.name}
          displayName={softwareTitle.display_name || softwareTitle.name}
          source={softwareTitle.source}
        />
      )}
      {showEditConfigurationModal && softwareInstaller && teamId && (
        <EditConfigurationModal
          softwareInstaller={softwareInstaller as IAppStoreApp}
          softwareId={softwareId}
          teamId={teamId}
          refetchSoftwareTitle={refetchSoftwareTitle}
          onExit={() => setShowEditConfigurationModal(false)}
        />
      )}
    </>
  );
};

export default SoftwareSummaryCard;
