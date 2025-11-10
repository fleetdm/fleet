/** software/titles/:id > Second section */

import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  ISoftwarePackage,
  IAppStoreApp,
  isSoftwarePackage,
} from "interfaces/software";
import { Platform } from "interfaces/platform";
import softwareAPI from "services/entities/software";

import { getSelfServiceTooltip } from "pages/SoftwarePage/helpers";

import Card from "components/Card";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Tag from "components/Tag";
import Button from "components/buttons/Button";

import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import CustomLink from "components/CustomLink";
import InstallerDetailsWidget from "pages/SoftwarePage/SoftwareTitleDetailsPage/SoftwareInstallerCard/InstallerDetailsWidget";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import EditSoftwareModal from "../EditSoftwareModal";
import ViewYamlModal from "../ViewYamlModal";

import {
  APP_STORE_APP_ACTION_OPTIONS,
  SOFTWARE_PACKAGE_ACTION_OPTIONS,
  downloadFile,
} from "./helpers";
import InstallerStatusTable from "./InstallerStatusTable";
import InstallerPoliciesTable from "./InstallerPoliciesTable";

const baseClass = "software-installer-card";

interface IStatusDisplayOption {
  displayName: string;
  iconName: "success" | "pending-outline" | "error";
  tooltip: React.ReactNode;
}

interface IActionsDropdownProps {
  installerType: "package" | "vpp";
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onEditSoftwareClick: () => void;
  gitOpsModeEnabled?: boolean;
  repoURL?: string;
  isFMA?: boolean;
}

export const SoftwareActionButtons = ({
  installerType,
  onDownloadClick,
  onDeleteClick,
  onEditSoftwareClick,
  gitOpsModeEnabled,
  repoURL,
  isFMA,
}: IActionsDropdownProps) => {
  let options =
    installerType === "package"
      ? [...SOFTWARE_PACKAGE_ACTION_OPTIONS]
      : [...APP_STORE_APP_ACTION_OPTIONS];

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
      // edit is disabled in gitOpsMode for VPP only
      // delete is disabled in gitOpsMode for software types that can't be added in GitOps mode (FMA, VPP)
      if (
        (option.value === "edit" && installerType === "vpp") ||
        (option.value === "delete" && (installerType === "vpp" || isFMA))
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
    edit: onEditSoftwareClick,
  };

  return (
    <div className={`${baseClass}__actions`}>
      {options.map((option) => {
        const ButtonContent = (
          <Button
            key={option.value}
            className={`btn btn-link ${baseClass}__action-btn`}
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
  softwareTitleName: string;
  isScriptPackage?: boolean;
  isIosOrIpadosApp?: boolean;
  name: string;
  version: string | null;
  addedTimestamp: string;
  status: {
    installed: number;
    pending: number;
    failed: number;
  };
  isSelfService: boolean;
  softwareId: number;
  iconUrl?: string | null;
  teamId: number;
  teamIdForApi?: number;
  softwareInstaller: ISoftwarePackage | IAppStoreApp;
  onDelete: () => void;
  refetchSoftwareTitle: () => void;
  isLoading: boolean;
  router: InjectedRouter;
  gitOpsYamlParam?: boolean;
}

// NOTE: This component is dependent on having either a software package
// (ISoftwarePackage) or an app store app (IAppStoreApp). If we add more types
// of packages we should consider refactoring this to be more dynamic.
const SoftwareInstallerCard = ({
  softwareTitleName,
  isScriptPackage = false,
  isIosOrIpadosApp = false,
  name,
  version,
  addedTimestamp,
  status,
  isSelfService,
  softwareInstaller,
  softwareId,
  iconUrl,
  teamId,
  teamIdForApi,
  onDelete,
  refetchSoftwareTitle,
  isLoading,
  router,
  gitOpsYamlParam = false,
}: ISoftwareInstallerCardProps) => {
  const installerType = isSoftwarePackage(softwareInstaller)
    ? "package"
    : "vpp";
  const isFleetMaintainedApp =
    "fleet_maintained_app_id" in softwareInstaller &&
    !!softwareInstaller.fleet_maintained_app_id;
  const isCustomPackage = installerType === "package" && !isFleetMaintainedApp;
  const sha256 =
    "hash_sha256" in softwareInstaller
      ? softwareInstaller.hash_sha256
      : undefined;

  const {
    automatic_install_policies: automaticInstallPolicies,
  } = softwareInstaller;

  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    config,
  } = useContext(AppContext);

  const { gitops_mode_enabled: gitOpsModeEnabled, repository_url: repoURL } =
    config?.gitops || {};

  const { renderFlash } = useContext(NotificationContext);

  // gitOpsYamlParam URL Param controls whether the View Yaml modal is opened on page load
  // as it automatically opens from adding flow of custom software in gitOps mode
  const [showViewYamlModal, setShowViewYamlModal] = useState(gitOpsYamlParam);
  const [showEditSoftwareModal, setShowEditSoftwareModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const onEditSoftwareClick = () => {
    setShowEditSoftwareModal(true);
  };

  const onDeleteClick = () => {
    setShowDeleteModal(true);
  };

  const onToggleViewYaml = () => {
    setShowViewYamlModal(!showViewYamlModal);
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
                  tipContent={getSelfServiceTooltip(isIosOrIpadosApp)}
                  underline={false}
                >
                  <Tag icon="user" text="Self-service" />
                </TooltipWrapper>
              )}
            </div>
          </div>
          <div className={`${baseClass}__actions-wrapper`}>
            {showActions && (
              <SoftwareActionButtons
                installerType={installerType}
                onDownloadClick={onDownloadClick}
                onDeleteClick={onDeleteClick}
                onEditSoftwareClick={onEditSoftwareClick}
                gitOpsModeEnabled={gitOpsModeEnabled}
                repoURL={repoURL}
                isFMA={isFleetMaintainedApp}
              />
            )}
          </div>
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
      {showEditSoftwareModal && (
        <EditSoftwareModal
          router={router}
          gitOpsModeEnabled={gitOpsModeEnabled}
          softwareId={softwareId}
          teamId={teamId}
          software={softwareInstaller}
          onExit={() => setShowEditSoftwareModal(false)}
          refetchSoftwareTitle={refetchSoftwareTitle}
          installerType={installerType}
          openViewYamlModal={onToggleViewYaml}
          isIosOrIpadosApp={isIosOrIpadosApp}
        />
      )}
      {showDeleteModal && (
        <DeleteSoftwareModal
          gitOpsModeEnabled={gitOpsModeEnabled}
          softwareId={softwareId}
          softwareInstallerName={softwareInstaller?.name}
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
          softwarePackage={softwareInstaller as ISoftwarePackage}
          onExit={onToggleViewYaml}
          isScriptPackage={isScriptPackage}
        />
      )}
    </Card>
  );
};

export default SoftwareInstallerCard;
