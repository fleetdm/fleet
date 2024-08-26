import React, {
  useCallback,
  useContext,
  useLayoutEffect,
  useState,
} from "react";
import FileSaver from "file-saver";
import { parse } from "content-disposition";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { SoftwareInstallStatus, ISoftwarePackage } from "interfaces/software";
import softwareAPI from "services/entities/software";

import { buildQueryStringFromParams } from "utilities/url";
import { internationalTimeFormat } from "utilities/helpers";
import { uploadedFromNow } from "utilities/date_format";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Card from "components/Card";
import Graphic from "components/Graphic";
import TooltipWrapper from "components/TooltipWrapper";
import DataSet from "components/DataSet";
import Icon from "components/Icon";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import AdvancedOptionsModal from "../AdvancedOptionsModal";
import {
  APP_STORE_APP_DROPDOWN_OPTIONS,
  SOFTWARE_PACAKGE_DROPDOWN_OPTIONS,
  downloadFile,
} from "./helpers";

const baseClass = "software-package-card";

/** TODO: pull this hook and SoftwareName component out. We could use this other places */
function useTruncatedElement<T extends HTMLElement>(ref: React.RefObject<T>) {
  const [isTruncated, setIsTruncated] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    if (element) {
      const { scrollWidth, clientWidth } = element;
      setIsTruncated(scrollWidth > clientWidth);
    }
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

const STATUS_DISPLAY_OPTIONS: Record<
  SoftwareInstallStatus,
  IStatusDisplayOption
> = {
  installed: {
    displayName: "Installed",
    iconName: "success",
    tooltip: (
      <>
        Fleet installed software on these hosts. Currently, if the software is
        uninstalled, the &quot;Installed&quot; status won&apos;t be updated.
      </>
    ),
  },
  pending: {
    displayName: "Pending",
    iconName: "pending-outline",
    tooltip: "Fleet will install software when these hosts come online.",
  },
  failed: {
    displayName: "Failed",
    iconName: "error",
    tooltip: "Fleet failed to install software on these hosts.",
  },
};

interface IPackageStatusCountProps {
  softwareId: number;
  status: SoftwareInstallStatus;
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
      title={
        <TooltipWrapper
          position="top"
          tipContent={displayData.tooltip}
          underline={false}
          showArrow
        >
          <div className={`${baseClass}__status-title`}>
            <Icon name={displayData.iconName} />
            <span>{displayData.displayName}</span>
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
  onAdvancedOptionsClick: () => void;
}

const ActionsDropdown = ({
  isSoftwarePackage,
  onDownloadClick,
  onDeleteClick,
  onAdvancedOptionsClick,
}: IActionsDropdownProps) => {
  const onSelect = (value: string) => {
    switch (value) {
      case "download":
        onDownloadClick();
        break;
      case "delete":
        onDeleteClick();
        break;
      case "advanced":
        onAdvancedOptionsClick();
        break;
      default:
      // noop
    }
  };

  return (
    <div className={`${baseClass}__actions`}>
      <Dropdown
        className={`${baseClass}__host-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        searchable={false}
        options={
          isSoftwarePackage
            ? SOFTWARE_PACAKGE_DROPDOWN_OPTIONS
            : APP_STORE_APP_DROPDOWN_OPTIONS
        }
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
}

// NOTE: This component is depeent on having either a software package
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
}: ISoftwarePackageCardProps) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [showAdvancedOptionsModal, setShowAdvancedOptionsModal] = useState(
    false
  );
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const onAdvancedOptionsClick = () => {
    setShowAdvancedOptionsModal(true);
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
      <div className={`${baseClass}__main-content`}>
        {/* TODO: main-info could be a seperate component as its reused on a couple
        pages already. Come back and pull this into a component */}
        <div className={`${baseClass}__main-info`}>
          {renderIcon()}
          <div className={`${baseClass}__info`}>
            <SoftwareName name={name} />
            <span className={`${baseClass}__details`}>{renderDetails()}</span>
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
      </div>
      <div className={`${baseClass}__actions-wrapper`}>
        {isSelfService && (
          <div className={`${baseClass}__self-service-badge`}>
            <Icon
              name="install-self-service"
              size="small"
              color="ui-fleet-black-75"
            />
            Self-service
          </div>
        )}
        {showActions && (
          <ActionsDropdown
            isSoftwarePackage={!!softwarePackage}
            onDownloadClick={onDownloadClick}
            onDeleteClick={onDeleteClick}
            onAdvancedOptionsClick={onAdvancedOptionsClick}
          />
        )}
      </div>
      {showAdvancedOptionsModal && (
        <AdvancedOptionsModal
          installScript={softwarePackage?.install_script ?? ""}
          preInstallQuery={softwarePackage?.pre_install_query}
          postInstallScript={softwarePackage?.post_install_script}
          onExit={() => setShowAdvancedOptionsModal(false)}
        />
      )}
      {showDeleteModal && (
        <DeleteSoftwareModal
          softwareId={softwareId}
          teamId={teamId}
          onExit={() => setShowDeleteModal(false)}
          onSuccess={onDeleteSuccess}
        />
      )}
    </Card>
  );
};

export default SoftwarePackageCard;
