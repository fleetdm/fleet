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
import softwareAPI from "services/entities/software";
import PATHS from "router/paths";

import { SELF_SERVICE_TOOLTIP } from "pages/SoftwarePage/helpers";

import Card from "components/Card";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Tag from "components/Tag";
import Button from "components/buttons/Button";

import { getPathWithQueryParams, QueryParams } from "utilities/url";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import CustomLink from "components/CustomLink";
import InstallerDetailsWidget from "pages/SoftwarePage/SoftwareTitleDetailsPage/SoftwareInstallerCard/InstallerDetailsWidget";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

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

// "pending" and "failed" each encompass both "_install" and "_uninstall" sub-statuses
type SoftwareInstallDisplayStatus = "installed" | "pending" | "failed";

const STATUS_DISPLAY_OPTIONS: Record<
  SoftwareInstallDisplayStatus,
  IStatusDisplayOption
> = {
  installed: {
    displayName: "Installed",
    iconName: "success",
    tooltip: (
      <>
        Software was installed on these hosts (install script finished
        <br />
        with exit code 0). Currently, if the software is uninstalled, the
        <br />
        &quot;Installed&quot; status won&apos;t be updated.
      </>
    ),
  },
  pending: {
    displayName: "Pending",
    iconName: "pending-outline",
    tooltip: (
      <>
        Fleet is installing/uninstalling or will
        <br />
        do so when the host comes online.
      </>
    ),
  },
  failed: {
    displayName: "Failed",
    iconName: "error",
    tooltip: (
      <>
        These hosts failed to install/uninstall software.
        <br />
        Click on a host to view error(s).
      </>
    ),
  },
};

interface IActionsDropdownProps {
  installerType: "package" | "vpp";
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onEditSoftwareClick: () => void;
  gitOpsModeEnabled?: boolean;
  repoURL?: string;
}

const SoftwareActionButtons = ({
  installerType,
  onDownloadClick,
  onDeleteClick,
  onEditSoftwareClick,
  gitOpsModeEnabled,
  repoURL,
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
      // delete is disabled in gitOpsMode for all installers (FMA, VPP, & custom packages)
      if (
        (option.value === "edit" && installerType === "vpp") ||
        option.value === "delete"
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
            <Icon name={option.iconName} color="core-fleet-blue" />
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
  name,
  version,
  addedTimestamp,
  status,
  isSelfService,
  softwareInstaller,
  softwareId,
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

  const [showEditSoftwareModal, setShowEditSoftwareModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const onEditSoftwareClick = () => {
    setShowEditSoftwareModal(true);
  };

  const onDeleteClick = () => {
    setShowDeleteModal(true);
  };

  // gitOpsYamlParam URL Param controls whether the View Yaml modal is opened
  // as it automatically opens from adding/editing flow of custom software in gitOps mode
  const onToggleViewYaml = () => {
    const newQueryParams: QueryParams = {
      team_id: teamId,
      gitops_yaml: !gitOpsYamlParam ? "true" : undefined,
    };

    router.push(
      getPathWithQueryParams(
        PATHS.SOFTWARE_TITLE_DETAILS(softwareId.toString()),
        newQueryParams
      )
    );
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

  let versionInfo = <span>{version}</span>;

  if (installerType === "vpp") {
    versionInfo = (
      <TooltipWrapper tipContent={<span>Updated every hour.</span>}>
        <span>{version}</span>
      </TooltipWrapper>
    );
  }

  if (installerType === "package" && !version) {
    versionInfo = (
      <TooltipWrapper
        tipContent={
          <span>
            Fleet couldn&apos;t read the version from {name}.{" "}
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
              text="Learn more"
              variant="tooltip-link"
            />
          </span>
        }
      >
        <span>Version (unknown)</span>
      </TooltipWrapper>
    );
  }

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  return (
    <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
      <div className={`${baseClass}__installer-header`}>
        <div className={`${baseClass}__row-1`}>
          <div className={`${baseClass}__row-1--responsive-wrap`}>
            <InstallerDetailsWidget
              softwareName={softwareInstaller?.name || name}
              installerType={installerType}
              versionInfo={versionInfo}
              addedTimestamp={addedTimestamp}
              sha256={sha256}
              isFma={isFleetMaintainedApp}
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
                  tipContent={SELF_SERVICE_TOOLTIP}
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
              />
            )}
          </div>
        </div>
        <div className={`${baseClass}__row-2`}>
          {gitOpsModeEnabled && isCustomPackage && (
            <div className={`${baseClass}__yaml-button-wrapper`}>
              <Button onClick={onToggleViewYaml}>View YAML</Button>
            </div>
          )}
        </div>
      </div>
      <div className={`${baseClass}__installer-status-table`}>
        <InstallerStatusTable
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
        />
      )}
      {showDeleteModal && (
        <DeleteSoftwareModal
          softwareId={softwareId}
          softwareInstallerName={softwareInstaller?.name}
          teamId={teamId}
          onExit={() => setShowDeleteModal(false)}
          onSuccess={onDeleteSuccess}
        />
      )}
      {gitOpsYamlParam && isCustomPackage && (
        <ViewYamlModal
          softwareTitleName={softwareTitleName}
          softwarePackage={softwareInstaller as ISoftwarePackage}
          onExit={onToggleViewYaml}
        />
      )}
    </Card>
  );
};

export default SoftwareInstallerCard;
