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

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import AdvancedOptionsModal from "../AdvancedOptionsModal";

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
  tooltip: string;
}

const STATUS_DISPLAY_OPTIONS: Record<
  SoftwareInstallStatus,
  IStatusDisplayOption
> = {
  installed: {
    displayName: "Installed",
    iconName: "success",
    tooltip: "Fleet installed software on these hosts.",
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

const DROPDOWN_OPTIONS = [
  {
    label: "Download",
    value: "download",
  },
  {
    label: "Delete",
    value: "delete",
  },
  {
    label: "Advanced options",
    value: "advanced",
  },
] as const;

const ActionsDropdown = ({
  onDownloadClick,
  onDeleteClick,
  onAdvancedOptionsClick,
}: {
  onDownloadClick: () => void;
  onDeleteClick: () => void;
  onAdvancedOptionsClick: () => void;
}) => {
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
        options={DROPDOWN_OPTIONS}
      />
    </div>
  );
};

interface ISoftwarePackageCardProps {
  softwarePackage: ISoftwarePackage;
  softwareId: number;
  teamId: number;
  onDelete: () => void;
}

const SoftwarePackageCard = ({
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
      const filename = softwarePackage.name;
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
      console.log(e);
      renderFlash("error", "Couldn’t download. Please try again.");
    }
  }, [renderFlash, softwareId, softwarePackage.name, teamId]);

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  return (
    <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
      <div className={`${baseClass}__main-content`}>
        {/* TODO: main-info could be a seperate component as its reused on a couple
        pages already. Come back and pull this into a component */}
        <div className={`${baseClass}__main-info`}>
          <Graphic name="file-pkg" />
          <div className={`${baseClass}__info`}>
            <SoftwareName name={softwarePackage.name} />
            <span className={`${baseClass}__details`}>
              <span>Version {softwarePackage.version} &bull; </span>
              <TooltipWrapper
                tipContent={internationalTimeFormat(
                  new Date(softwarePackage.uploaded_at)
                )}
                underline={false}
              >
                {uploadedFromNow(softwarePackage.uploaded_at)}
              </TooltipWrapper>
            </span>
          </div>
        </div>
        <div className={`${baseClass}__package-statuses`}>
          <PackageStatusCount
            softwareId={softwareId}
            status="installed"
            count={softwarePackage.status.installed}
            teamId={teamId}
          />
          <PackageStatusCount
            softwareId={softwareId}
            status="pending"
            count={softwarePackage.status.pending}
            teamId={teamId}
          />
          <PackageStatusCount
            softwareId={softwareId}
            status="failed"
            count={softwarePackage.status.failed}
            teamId={teamId}
          />
        </div>
      </div>
      <div className={`${baseClass}__actions-wrapper`}>
        {softwarePackage.self_service && (
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
            onDownloadClick={onDownloadClick}
            onDeleteClick={onDeleteClick}
            onAdvancedOptionsClick={onAdvancedOptionsClick}
          />
        )}
      </div>
      {showAdvancedOptionsModal && (
        <AdvancedOptionsModal
          installScript={softwarePackage.install_script}
          preInstallQuery={softwarePackage.pre_install_query}
          postInstallScript={softwarePackage.post_install_script}
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
