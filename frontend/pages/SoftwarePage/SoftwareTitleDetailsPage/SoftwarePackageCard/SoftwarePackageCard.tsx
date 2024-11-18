import React, {
  useCallback,
  useContext,
  useLayoutEffect,
  useState,
} from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ISoftwarePackage } from "interfaces/software";
import softwareAPI from "services/entities/software";

import { buildQueryStringFromParams } from "utilities/url";
import { internationalTimeFormat } from "utilities/helpers";
import { uploadedFromNow } from "utilities/date_format";

import Card from "components/Card";
import Graphic from "components/Graphic";
import ActionsDropdown from "components/ActionsDropdown";
import TooltipWrapper from "components/TooltipWrapper";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import Tag from "components/Tag";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import EditSoftwareModal from "../EditSoftwareModal";
import {
  APP_STORE_APP_DROPDOWN_OPTIONS,
  SOFTWARE_PACKAGE_DROPDOWN_OPTIONS,
  downloadFile,
} from "./helpers";
import AutomaticInstallModal from "../AutomaticInstallModal";

const baseClass = "software-package-card";

/** TODO: pull this hook and SoftwareName component out. We could use this other places */
function useTruncatedElement<T extends HTMLElement>(ref: React.RefObject<T>) {
  const [isTruncated, setIsTruncated] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    function updateIsTruncated() {
      if (element) {
        const { scrollWidth, clientWidth } = element;
        setIsTruncated(scrollWidth > clientWidth);
      }
    }
    window.addEventListener("resize", updateIsTruncated);
    updateIsTruncated();
    return () => window.removeEventListener("resize", updateIsTruncated);
  }, [ref]);

  return isTruncated;
}

interface ISoftwareNameProps {
  name: string;
}

const SoftwareName = ({ name }: ISoftwareNameProps) => {
  const titleRef = React.useRef<HTMLDivElement>(null);
  const isTruncated = useTruncatedElement(titleRef);

  return (
    <TooltipWrapper
      tipContent={name}
      position="top"
      underline={false}
      disableTooltip={!isTruncated}
      showArrow
    >
      <div ref={titleRef} className={`${baseClass}__title`}>
        {name}
      </div>
    </TooltipWrapper>
  );
};

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

interface IPackageStatusCountProps {
  softwareId: number;
  status: SoftwareInstallDisplayStatus;
  count: number;
  teamId?: number;
}

const PackageStatusCount = ({
  softwareId,
  status,
  count,
  teamId,
}: IPackageStatusCountProps) => {
  const displayData = STATUS_DISPLAY_OPTIONS[status];
  const linkUrl = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams({
    software_title_id: softwareId,
    software_status: status,
    team_id: teamId,
  })}`;
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
  isSoftwarePackage: boolean;
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onEditSoftwareClick: () => void;
}

const SoftwareActionsDropdown = ({
  isSoftwarePackage,
  onDownloadClick,
  onDeleteClick,
  onEditSoftwareClick,
}: IActionsDropdownProps) => {
  const onSelect = (value: string) => {
    switch (value) {
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

  return (
    <div className={`${baseClass}__actions`}>
      <ActionsDropdown
        className={`${baseClass}__host-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        isSearchable={false}
        options={
          isSoftwarePackage
            ? [...SOFTWARE_PACKAGE_DROPDOWN_OPTIONS]
            : [...APP_STORE_APP_DROPDOWN_OPTIONS]
        }
        menuAlign="right"
      />
    </div>
  );
};

interface ISoftwarePackageCardProps {
  name: string;
  version: string;
  uploadedAt: string; // TODO: optional?
  status: {
    installed: number;
    pending: number;
    failed: number;
  };
  isSelfService: boolean;
  softwareId: number;
  teamId: number;
  // NOTE: we will only have this if we are working with a software package.
  softwarePackage?: ISoftwarePackage;
  onDelete: () => void;
  router: InjectedRouter;
  refetchSoftwareTitle: () => void;
}

// NOTE: This component is dependent on having either a software package
// (ISoftwarePackage) or an app store app (IAppStoreApp). If we add more types
// of packages we should consider refactoring this to be more dynamic.
const SoftwarePackageCard = ({
  name,
  version,
  uploadedAt,
  status,
  isSelfService,
  softwarePackage,
  softwareId,
  teamId,
  onDelete,
  router,
  refetchSoftwareTitle,
}: ISoftwarePackageCardProps) => {
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

  const renderIcon = () => {
    return softwarePackage ? (
      <Graphic name="file-pkg" />
    ) : (
      <SoftwareIcon name="appStore" size="medium" />
    );
  };

  const renderDetails = () => {
    return !uploadedAt ? (
      <span>Version {version}</span>
    ) : (
      <>
        <span>Version {version} &bull; </span>
        <TooltipWrapper
          tipContent={internationalTimeFormat(new Date(uploadedAt))}
          underline={false}
        >
          {uploadedFromNow(uploadedAt)}
        </TooltipWrapper>
      </>
    );
  };

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  return (
    <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
      <div className={`${baseClass}__row-1`}>
        {/* TODO: main-info could be a seperate component as its reused on a couple
        pages already. Come back and pull this into a component */}
        <div className={`${baseClass}__main-info`}>
          {renderIcon()}
          <div className={`${baseClass}__info`}>
            <SoftwareName name={softwarePackage?.name || name} />
            <span className={`${baseClass}__details`}>{renderDetails()}</span>
          </div>
        </div>
        <div className={`${baseClass}__actions-wrapper`}>
          {softwarePackage?.automatic_install_policies && (
            <TooltipWrapper
              showArrow
              position="top"
              tipContent="Click to see policy that triggers automatic install."
              underline={false}
            >
              <Tag
                icon="refresh"
                text="Automatic install"
                onClick={() => console.log("test")}
              />
            </TooltipWrapper>
          )}
          {isSelfService && <Tag icon="user" text="Self-service" />}
          {showActions && (
            <SoftwareActionsDropdown
              isSoftwarePackage={!!softwarePackage}
              onDownloadClick={onDownloadClick}
              onDeleteClick={onDeleteClick}
              onEditSoftwareClick={onEditSoftwareClick}
            />
          )}
        </div>
      </div>
      <div className={`${baseClass}__package-statuses`}>
        <PackageStatusCount
          softwareId={softwareId}
          status="installed"
          count={status.installed}
          teamId={teamId}
        />
        <PackageStatusCount
          softwareId={softwareId}
          status="pending"
          count={status.pending}
          teamId={teamId}
        />
        <PackageStatusCount
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
          software={softwarePackage}
          onExit={() => setShowEditSoftwareModal(false)}
          router={router}
          refetchSoftwareTitle={refetchSoftwareTitle}
        />
      )}
      {showDeleteModal && (
        <DeleteSoftwareModal
          softwareId={softwareId}
          softwarePackageName={softwarePackage?.name}
          teamId={teamId}
          onExit={() => setShowDeleteModal(false)}
          onSuccess={onDeleteSuccess}
        />
      )}
      {true && softwarePackage && (
        <AutomaticInstallModal
          policies={softwarePackage.automatic_install_policies}
          onExit={() => setShowAutomaticInstallModal(false)}
        />
      )}
    </Card>
  );
};

export default SoftwarePackageCard;
