/** software/titles/:id > Second section */

import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  ISoftwareTitleDetails,
  ISoftwarePackage,
  InstallerType,
} from "interfaces/software";
import softwareAPI from "services/entities/software";

import { useSoftwareInstaller } from "hooks/useSoftwareInstallerMeta";

import {
  getSelfServiceTooltip,
  getAutoUpdatesTooltip,
} from "pages/SoftwarePage/helpers";

import Card from "components/Card";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Tag from "components/Tag";
import Button from "components/buttons/Button";

import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import CustomLink from "components/CustomLink";
import InstallerDetailsWidget from "pages/SoftwarePage/SoftwareTitleDetailsPage/SoftwareInstallerCard/InstallerDetailsWidget";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import ViewYamlModal from "../ViewYamlModal";

import {
  ANDROID_PLAY_STORE_APP_ACTION_OPTIONS,
  APP_STORE_APP_ACTION_OPTIONS,
  SOFTWARE_PACKAGE_ACTION_OPTIONS,
  downloadFile,
} from "./helpers";
import InstallerStatusTable from "./InstallerStatusTable";
import InstallerPoliciesTable from "./InstallerPoliciesTable";

const baseClass = "software-installer-card";

interface IActionsDropdownProps {
  installerType: InstallerType;
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  gitOpsModeEnabled?: boolean;
  repoURL?: string;
  isFMA?: boolean;
  isAndroidPlayStoreApp?: boolean;
}

export const SoftwareActionButtons = ({
  installerType,
  onDownloadClick,
  onDeleteClick,
  gitOpsModeEnabled,
  repoURL,
  isFMA,
  isAndroidPlayStoreApp,
}: IActionsDropdownProps) => {
  let options = [...SOFTWARE_PACKAGE_ACTION_OPTIONS];

  if (installerType === "app-store") {
    options = isAndroidPlayStoreApp
      ? [...ANDROID_PLAY_STORE_APP_ACTION_OPTIONS]
      : [...APP_STORE_APP_ACTION_OPTIONS];
  }

  if (gitOpsModeEnabled) {
    const tooltipContent = (
      <>
        {repoURL && (
          <>
            Manage in{" "}
            <CustomLink
              newTab
              text="YAML"
              variant="tooltip-link"
              url={repoURL}
            />
            <br />
          </>
        )}
        (GitOps mode enabled)
      </>
    );
    options = options.map((option) => {
      // delete is disabled in gitOpsMode for software types that can't be added in GitOps mode (FMA, VPP)
      if (
        option.value === "delete" &&
        (installerType === "app-store" || isFMA)
      ) {
        return {
          ...option,
          disabled: true,
          tooltipContent,
        };
      }
      return option;
    });
  }

  // Map action values to handlers
  const actionHandlers = {
    download: onDownloadClick,
    delete: onDeleteClick,
  };

  return (
    <div className={`${baseClass}__actions-wrapper`}>
      {options.map((option) => {
        const ButtonContent = (
          <Button
            key={option.value}
            className={`${baseClass}__action-btn`}
            disabled={option.disabled}
            onClick={() =>
              actionHandlers[option.value as keyof typeof actionHandlers]?.()
            }
            variant="icon"
          >
            <Icon name={option.iconName} color="ui-fleet-black-75" />
          </Button>
        );

        // If there's a tooltip, wrap the button
        return option.tooltipContent ? (
          <TooltipWrapper
            key={option.value}
            tipContent={option.tooltipContent}
            underline={false}
          >
            {ButtonContent}
          </TooltipWrapper>
        ) : (
          ButtonContent
        );
      })}
    </div>
  );
};

interface ISoftwareInstallerCardProps {
  softwareId: number;
  teamId: number;
  teamIdForApi?: number;
  onDelete: () => void;
  isLoading: boolean;
  onToggleViewYaml: () => void;
  showViewYamlModal: boolean;
  softwareTitle: ISoftwareTitleDetails;
}

// NOTE: This component is dependent on having either a software package
// (ISoftwarePackage) or an app store app (IAppStoreApp). If we add more types
// of packages we should consider refactoring this to be more dynamic.
const SoftwareInstallerCard = ({
  softwareId,
  teamId,
  teamIdForApi,
  onDelete,
  isLoading,
  onToggleViewYaml,
  showViewYamlModal,
  softwareTitle,
}: ISoftwareInstallerCardProps) => {
  const softwareInstallerMetaData = useSoftwareInstaller(softwareTitle);

  if (!softwareInstallerMetaData) {
    // This should never happen for SoftwareInstallerCard; fail fast in dev.
    throw new Error(
      "useSoftwareInstaller: called with a softwareTitle that has no installer"
    );
  }

  const { cardInfo, meta: softwareInstallerMeta } = softwareInstallerMetaData;

  const {
    softwareTitleName,
    softwareDisplayName,
    softwareInstaller,
    name,
    version,
    addedTimestamp,
    status,
    iconUrl,
    displayName,
    isSelfService,
    isScriptPackage,
    autoUpdateEnabled,
    autoUpdateStartTime,
    autoUpdateEndTime,
  } = cardInfo;

  const {
    installerType,
    isAndroidPlayStoreApp,
    isFleetMaintainedApp,
    isCustomPackage,
    isIosOrIpadosApp,
    sha256,
    androidPlayStoreId,
    automaticInstallPolicies,
    gitOpsModeEnabled,
    repoURL,
  } = softwareInstallerMeta;

  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const onDeleteClick = () => {
    setShowDeleteModal(true);
  };

  const onDeleteSuccess = useCallback(() => {
    setShowDeleteModal(false);
    onDelete();
  }, [onDelete]);

  const onDownloadClick = useCallback(async () => {
    try {
      const resp = await softwareAPI.getSoftwarePackageToken(
        softwareId,
        teamId
      );
      if (!resp.token) {
        throw new Error("No download token returned");
      }
      // Now that we received the download token, we construct the download URL.
      const { origin } = global.window.location;
      const url = `${origin}${URL_PREFIX}/api${endpoints.SOFTWARE_PACKAGE_TOKEN(
        softwareId
      )}/${resp.token}`;
      // The download occurs without any additional authentication.
      downloadFile(url, name);
    } catch (e) {
      renderFlash("error", "Couldn't download. Please try again.");
    }
  }, [renderFlash, softwareId, name, teamId]);

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  return (
    <Card borderRadiusSize="xxlarge" className={baseClass}>
      <div className={`${baseClass}__installer-header`}>
        <div className={`${baseClass}__row-1`}>
          <div className={`${baseClass}__row-1--responsive-wrap`}>
            <InstallerDetailsWidget
              softwareName={softwareInstaller?.name || name}
              installerType={installerType}
              version={version}
              addedTimestamp={addedTimestamp}
              sha256={sha256}
              isFma={isFleetMaintainedApp}
              isScriptPackage={isScriptPackage}
              androidPlayStoreId={androidPlayStoreId}
            />
            <div className={`${baseClass}__tags-wrapper`}>
              {Array.isArray(automaticInstallPolicies) &&
                automaticInstallPolicies.length > 0 && (
                  <TooltipWrapper
                    showArrow
                    position="top"
                    tipContent={
                      automaticInstallPolicies.length === 1
                        ? "A policy triggers install."
                        : `${automaticInstallPolicies.length} policies trigger install.`
                    }
                    underline={false}
                  >
                    <Tag icon="refresh" text="Automatic install" />
                  </TooltipWrapper>
                )}
              {isSelfService && (
                <TooltipWrapper
                  showArrow
                  position="top"
                  tipContent={getSelfServiceTooltip(
                    isIosOrIpadosApp,
                    isAndroidPlayStoreApp
                  )}
                  underline={false}
                >
                  <Tag icon="user" text="Self-service" />
                </TooltipWrapper>
              )}
              {autoUpdateEnabled && (
                <TooltipWrapper
                  className={`${baseClass}__auto-updates-tooltip`}
                  showArrow
                  position="top"
                  tipContent={getAutoUpdatesTooltip(
                    autoUpdateStartTime || "",
                    autoUpdateEndTime || ""
                  )}
                  underline={false}
                >
                  <Tag icon="clock" text="Auto updates" />
                </TooltipWrapper>
              )}
            </div>
          </div>
          {showActions && (
            <SoftwareActionButtons
              installerType={installerType}
              onDownloadClick={onDownloadClick}
              onDeleteClick={onDeleteClick}
              gitOpsModeEnabled={gitOpsModeEnabled}
              repoURL={repoURL}
              isFMA={isFleetMaintainedApp}
              isAndroidPlayStoreApp={isAndroidPlayStoreApp}
            />
          )}
        </div>
        {gitOpsModeEnabled && isCustomPackage && (
          <div className={`${baseClass}__row-2`}>
            <div className={`${baseClass}__yaml-button-wrapper`}>
              <Button onClick={onToggleViewYaml}>View YAML</Button>
            </div>
          </div>
        )}
      </div>
      <div className={`${baseClass}__installer-status-table`}>
        <InstallerStatusTable
          isScriptPackage={isScriptPackage}
          isAndroidPlayStoreApp={isAndroidPlayStoreApp}
          softwareId={softwareId}
          teamId={teamId}
          status={status}
          isLoading={isLoading}
        />
      </div>
      {automaticInstallPolicies && (
        <div className={`${baseClass}__installer-policies-table`}>
          <InstallerPoliciesTable
            teamId={teamId}
            isLoading={isLoading}
            policies={automaticInstallPolicies}
          />
        </div>
      )}
      {showDeleteModal && (
        <DeleteSoftwareModal
          gitOpsModeEnabled={gitOpsModeEnabled}
          softwareId={softwareId}
          softwareDisplayName={softwareDisplayName}
          softwareTitleName={softwareTitleName}
          teamId={teamId}
          onExit={() => setShowDeleteModal(false)}
          onSuccess={onDeleteSuccess}
        />
      )}
      {showViewYamlModal && isCustomPackage && (
        <ViewYamlModal
          softwareTitleName={softwareTitleName}
          softwareTitleId={softwareId}
          teamId={teamId}
          iconUrl={iconUrl}
          displayName={displayName}
          softwarePackage={softwareInstaller as ISoftwarePackage}
          onExit={onToggleViewYaml}
          isScriptPackage={isScriptPackage}
          isIosOrIpadosApp={isIosOrIpadosApp}
        />
      )}
    </Card>
  );
};

export default SoftwareInstallerCard;
