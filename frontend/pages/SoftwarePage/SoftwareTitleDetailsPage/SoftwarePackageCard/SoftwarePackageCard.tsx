import React, {
  useCallback,
  useContext,
  useLayoutEffect,
  useState,
} from "react";
import FileSaver from "file-saver";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import {
  SoftwareInstallStatus,
  ISoftwarePackage,
  ISoftwarePackageLabel,
} from "interfaces/software";
import softwareAPI from "services/entities/software";

import { buildQueryStringFromParams } from "utilities/url";
import {
  internationalTimeFormat,
  tooltipTextWithLineBreaks,
} from "utilities/helpers";
import { uploadedFromNow } from "utilities/date_format";
import strUtils from "utilities/strings";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Card from "components/Card";
import Graphic from "components/Graphic";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import PackageOptionsModal from "../PackageOptionsModal";
import {
  APP_STORE_APP_DROPDOWN_OPTIONS,
  SOFTWARE_PACKAGE_DROPDOWN_OPTIONS,
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
  const name2 = "really really really really really so so long title name";
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
        {name2}
      </div>
    </TooltipWrapper>
  );
};

interface IStatusDisplayOption {
  displayName: string;
  iconName:
    | "success"
    | "success-outline"
    | "pending-outline"
    | "disable"
    | "error";
  tooltip: React.ReactNode;
}

interface IPackageStatusCountProps {
  softwareId: number;
  status: SoftwareInstallStatus;
  count: number;
  teamId?: number;
  isAutomaticInstall?: boolean;
}

const PackageStatusCount = ({
  softwareId,
  status,
  count,
  teamId,
  isAutomaticInstall,
}: IPackageStatusCountProps) => {
  const STATUS_DISPLAY_OPTIONS: Record<
    SoftwareInstallStatus,
    IStatusDisplayOption
  > = {
    verified: {
      displayName: "Verified",
      iconName: "success",
      tooltip: <>Software is installed on these hosts. Fleet verified.</>,
    },
    verifying: {
      displayName: "Verifying",
      iconName: "success-outline", // TODO
      tooltip: (
        <>
          Software is installed on these hosts (install script exited with
          <br /> exit code: 0). Fleet is verifying.
        </>
      ),
    },
    pending: {
      displayName: "Pending",
      iconName: "pending-outline",
      tooltip: isAutomaticInstall ? (
        <>
          Checking if the software is missing or an older version is
          <br />
          installed. If it is, Fleet is installing or will install when the host
          <br />
          comes online.
        </>
      ) : (
        <>
          Fleet is installing or will install when <br />
          the host comes online.
        </>
      ),
    },
    blocked: {
      displayName: "Blocked",
      iconName: "disable",
      tooltip: (
        <>
          Pre-install condition wasn&apos;t met.
          <br /> The query didn&apos;t return results.
        </>
      ),
    },
    failed: {
      displayName: "Failed",
      iconName: "error",
      tooltip: (
        <>
          These hosts failed to install software.
          <br /> Click on a host to view error(s).
        </>
      ),
    },
  };

  const displayData = STATUS_DISPLAY_OPTIONS[status];
  const linkUrl = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams({
    software_title_id: softwareId,
    software_status: status,
    team_id: teamId,
  })}`;
  return (
    <div className={`${baseClass}__status`}>
      <TooltipWrapper
        position="top"
        tipContent={displayData.tooltip}
        underline={false}
        showArrow
        className={`${baseClass}__status-title`}
      >
        <Icon name={displayData.iconName} />
        <span>{displayData.displayName}</span>
      </TooltipWrapper>
      <a className={`${baseClass}__status-count`} href={linkUrl}>
        {count || 0} hosts
      </a>
    </div>
  );
};

interface IActionsDropdownProps {
  isSoftwarePackage: boolean;
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onOptionsClick: () => void;
}

const ActionsDropdown = ({
  isSoftwarePackage,
  onDownloadClick,
  onDeleteClick,
  onOptionsClick,
}: IActionsDropdownProps) => {
  const onSelect = (value: string) => {
    switch (value) {
      case "download":
        onDownloadClick();
        break;
      case "delete":
        onDeleteClick();
        break;
      case "options":
        onOptionsClick();
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
            ? SOFTWARE_PACKAGE_DROPDOWN_OPTIONS
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
    verified: number;
    verifying: number;
    pending: number;
    blocked: number;
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

  const [showOptionsModal, setShowOptionsModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const onOptionsClick = () => {
    setShowOptionsModal(true);
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
      const resp = await softwareAPI.downloadSoftwarePackage(
        softwareId,
        teamId
      );
      const contentLength = parseInt(resp.headers["content-length"], 10);
      if (contentLength !== resp.data.size) {
        throw new Error(
          `Byte size (${resp.data.size}) does not match content-length header (${contentLength})`
        );
      }
      const filename = name;
      const file = new File([resp.data], filename, {
        type: "application/octet-stream",
      });
      if (file.size === 0) {
        throw new Error("Downloaded file is empty");
      }
      if (file.size !== resp.data.size) {
        throw new Error(
          `File size (${file.size}) does not match expected size (${resp.data.size})`
        );
      }
      FileSaver.saveAs(file);
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

  const renderSelfServiceInfo = () => {
    return (
      <div className={`${baseClass}__badge`}>
        <Icon
          name="install-self-service"
          size="small"
          color="ui-fleet-black-75"
        />
        Self-service
      </div>
    );
  };

  const renderLabelInfo = () => {
    const softwarePackageTEST: {
      labels_include_any: ISoftwarePackageLabel[];
      labels_exclude_any: ISoftwarePackageLabel[];
    } = {
      labels_include_any: [],
      labels_exclude_any: [
        { name: "Cool label", id: 40 },
        { name: "Cooler label", id: 50 },
      ],
    };
    const labels = softwarePackageTEST?.labels_include_any?.length
      ? softwarePackageTEST.labels_include_any.map((label) => label.name)
      : softwarePackageTEST.labels_exclude_any.map((label) => label.name) || [];

    const count = labels.length;

    return (
      <TooltipWrapper
        tipContent={tooltipTextWithLineBreaks(labels)}
        underline={false}
        showArrow
        position="top"
        tipOffset={10}
      >
        <div className={`${baseClass}__badge`}>
          <Icon name="filter" size="small" color="ui-fleet-black-75" />
          {`${count} ${strUtils.pluralize(count, "label")}`}
        </div>
      </TooltipWrapper>
    );
  };

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  // const hasLabelInfo =
  //   (softwarePackage?.labels_include_any &&
  //     softwarePackage?.labels_include_any?.length > 0) ||
  //   (softwarePackage?.labels_exclude_any &&
  //     softwarePackage?.labels_exclude_any?.length > 0);

  const hasLabelInfo = true;
  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      className={baseClass}
      paddingSize="xxlarge"
    >
      <div className={`${baseClass}__header`}>
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
        </div>
        <div className={`${baseClass}__actions-wrapper`}>
          {isSelfService && renderSelfServiceInfo()}
          {hasLabelInfo && renderLabelInfo()}
          {showActions && (
            <ActionsDropdown
              isSoftwarePackage={!!softwarePackage}
              onDownloadClick={onDownloadClick}
              onDeleteClick={onDeleteClick}
              onOptionsClick={onOptionsClick}
            />
          )}
        </div>
      </div>
      <div className={`${baseClass}__package-statuses`}>
        <PackageStatusCount
          softwareId={softwareId}
          status="verified"
          count={status.verified}
          teamId={teamId}
        />
        <PackageStatusCount
          softwareId={softwareId}
          status="verifying"
          count={status.verifying}
          teamId={teamId}
        />
        <PackageStatusCount
          softwareId={softwareId}
          status="pending"
          count={status.pending}
          teamId={teamId}
          isAutomaticInstall={softwarePackage?.install === "automatic"}
        />
        {!!softwarePackage && (
          <PackageStatusCount
            softwareId={softwareId}
            status="blocked"
            count={status.blocked}
            teamId={teamId}
          />
        )}
        <PackageStatusCount
          softwareId={softwareId}
          status="failed"
          count={status.failed}
          teamId={teamId}
        />
      </div>
      {showOptionsModal && (
        <PackageOptionsModal
          installScript={softwarePackage?.install_script ?? ""}
          preInstallQuery={softwarePackage?.pre_install_query}
          postInstallScript={softwarePackage?.post_install_script}
          onExit={() => setShowOptionsModal(false)}
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
