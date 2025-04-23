/** software/titles/:id > Second section */

import React, { useCallback, useContext, useState } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  ISoftwarePackage,
  IAppStoreApp,
  isSoftwarePackage,
} from "interfaces/software";
import softwareAPI from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import { SELF_SERVICE_TOOLTIP } from "pages/SoftwarePage/helpers";

import Card from "components/Card";

import ActionsDropdown from "components/ActionsDropdown";
import TooltipWrapper from "components/TooltipWrapper";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import Tag from "components/Tag";

import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import CustomLink from "components/CustomLink";
import SoftwareDetailsWidget from "pages/SoftwarePage/components/SoftwareDetailsWidget";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import EditSoftwareModal from "../EditSoftwareModal";
import {
  APP_STORE_APP_DROPDOWN_OPTIONS,
  SOFTWARE_PACKAGE_DROPDOWN_OPTIONS,
  downloadFile,
} from "./helpers";
import AutomaticInstallModal from "../AutomaticInstallModal";

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
        Software is installed on these hosts (install script finished
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

interface IInstallerStatusCountProps {
  softwareId: number;
  status: SoftwareInstallDisplayStatus;
  count: number;
  teamId?: number;
}

const InstallerStatusCount = ({
  softwareId,
  status,
  count,
  teamId,
}: IInstallerStatusCountProps) => {
  const displayData = STATUS_DISPLAY_OPTIONS[status];
  const linkUrl = getPathWithQueryParams(PATHS.MANAGE_HOSTS, {
    software_title_id: softwareId,
    software_status: status,
    team_id: teamId,
  });

  return (
    <DataSet
      className={`${baseClass}__status`}
      title={
        <TooltipWrapper
          position="top"
          tipContent={displayData.tooltip}
          underline={false}
          showArrow
          tipOffset={10}
        >
          <div className={`${baseClass}__status-title`}>
            <Icon name={displayData.iconName} />
            <div>{displayData.displayName}</div>
          </div>
        </TooltipWrapper>
      }
      value={
        <a className={`${baseClass}__status-count`} href={linkUrl}>
          {count} hosts
        </a>
      }
    />
  );
};

interface IActionsDropdownProps {
  installerType: "package" | "vpp";
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onEditSoftwareClick: () => void;
}

const SoftwareActionsDropdown = ({
  installerType,
  onDownloadClick,
  onDeleteClick,
  onEditSoftwareClick,
}: IActionsDropdownProps) => {
  const config = useContext(AppContext).config;
  const { gitops_mode_enabled: gitOpsModeEnabled, repository_url: repoURL } =
    config?.gitops || {};

  const onSelect = (action: string) => {
    switch (action) {
      case "download":
        onDownloadClick();
        break;
      case "delete":
        onDeleteClick();
        break;
      case "edit":
        onEditSoftwareClick();
        break;
      default:
      // noop
    }
  };

  let options =
    installerType === "package"
      ? [...SOFTWARE_PACKAGE_DROPDOWN_OPTIONS]
      : [...APP_STORE_APP_DROPDOWN_OPTIONS];

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
      if (option.value === "edit" || option.value === "delete") {
        return {
          ...option,
          disabled: true,
          tooltipContent,
        };
      }
      return option;
    });
  }

  return (
    <div className={`${baseClass}__actions`}>
      <ActionsDropdown
        className={`${baseClass}__software-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        menuAlign="right"
        options={options}
      />
    </div>
  );
};

interface ISoftwareInstallerCardProps {
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
  softwareInstaller: ISoftwarePackage | IAppStoreApp;
  onDelete: () => void;
  refetchSoftwareTitle: () => void;
}

// NOTE: This component is dependent on having either a software package
// (ISoftwarePackage) or an app store app (IAppStoreApp). If we add more types
// of packages we should consider refactoring this to be more dynamic.
const SoftwareInstallerCard = ({
  name,
  version,
  addedTimestamp,
  status,
  isSelfService,
  softwareInstaller,
  softwareId,
  teamId,
  onDelete,
  refetchSoftwareTitle,
}: ISoftwareInstallerCardProps) => {
  const installerType = isSoftwarePackage(softwareInstaller)
    ? "package"
    : "vpp";
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const [showEditSoftwareModal, setShowEditSoftwareModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAutomaticInstallModal, setShowAutomaticInstallModal] = useState(
    false
  );

  const onEditSoftwareClick = () => {
    setShowEditSoftwareModal(true);
  };

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
      <div className={`${baseClass}__row-1`}>
        <div className={`${baseClass}__row-1--responsive`}>
          <SoftwareDetailsWidget
            softwareName={softwareInstaller?.name || name}
            installerType={installerType}
            versionInfo={versionInfo}
            addedTimestamp={addedTimestamp}
          />
          <div className={`${baseClass}__tags-wrapper`}>
            {Array.isArray(softwareInstaller.automatic_install_policies) &&
              softwareInstaller.automatic_install_policies.length > 0 && (
                <TooltipWrapper
                  showArrow
                  position="top"
                  tipContent="Click to see policy that triggers automatic install."
                  underline={false}
                >
                  <Tag
                    icon="refresh"
                    text="Automatic install"
                    onClick={() => setShowAutomaticInstallModal(true)}
                  />
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
            <SoftwareActionsDropdown
              installerType={installerType}
              onDownloadClick={onDownloadClick}
              onDeleteClick={onDeleteClick}
              onEditSoftwareClick={onEditSoftwareClick}
            />
          )}
        </div>
      </div>
      <div className={`${baseClass}__installer-statuses`}>
        <InstallerStatusCount
          softwareId={softwareId}
          status="installed"
          count={status.installed}
          teamId={teamId}
        />
        <InstallerStatusCount
          softwareId={softwareId}
          status="pending"
          count={status.pending}
          teamId={teamId}
        />
        <InstallerStatusCount
          softwareId={softwareId}
          status="failed"
          count={status.failed}
          teamId={teamId}
        />
      </div>
      {showEditSoftwareModal && (
        <EditSoftwareModal
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
      {showAutomaticInstallModal &&
        softwareInstaller?.automatic_install_policies &&
        softwareInstaller?.automatic_install_policies.length > 0 && (
          <AutomaticInstallModal
            teamId={teamId}
            policies={softwareInstaller.automatic_install_policies}
            onExit={() => setShowAutomaticInstallModal(false)}
          />
        )}
    </Card>
  );
};

export default SoftwareInstallerCard;
