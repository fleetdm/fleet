import React, { useContext, useState } from "react";

import endpoints from "utilities/endpoints";
import { ISoftwareInstallStatus, ISoftwarePackage } from "interfaces/software";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import { buildQueryStringFromParams } from "utilities/url";
import { internationalTimeFormat } from "utilities/helpers";
import { uploadedFromNow } from "utilities/date_format";

import Card from "components/Card";
import Graphic from "components/Graphic";
import TooltipWrapper from "components/TooltipWrapper";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import Button from "components/buttons/Button";

import DeleteSoftwareModal from "../DeleteSoftwareModal";
import AdvancedOptionsModal from "../AdvancedOptionsModal";

const baseClass = "software-package-card";

interface IStatusDisplayOption {
  displayName: string;
  iconName: "success" | "pending-outline" | "error";
  tooltip: string;
}

const STATUS_DISPLAY_OPTIONS: Record<
  ISoftwareInstallStatus,
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
  status: ISoftwareInstallStatus;
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
    software_title_status: status,
    team_id: teamId,
  })}`;
  return (
    <DataSet
      title={
        <TooltipWrapper
          position="top"
          tipContent={displayData.tooltip}
          underline={false}
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

interface ISoftwarePackageCardProps {
  softwarePackage: ISoftwarePackage;
  softwareId: number;
  teamId: number;
}

const SoftwarePackageCard = ({
  softwarePackage,
  softwareId,
  teamId,
}: ISoftwarePackageCardProps) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

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

  const showActions =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  const downloadUrl = `/api${endpoints.SOFTWARE_PACKAGE(
    softwareId
  )}?${buildQueryStringFromParams({ alt: "media", team_id: teamId })}`;

  return (
    <Card borderRadiusSize="large" includeShadow className={baseClass}>
      <div className={`${baseClass}__main-content`}>
        {/* TODO: main-info could be a seperate component as its reused on a couple
        pages already. Come back and pull this into a component */}
        <div className={`${baseClass}__main-info`}>
          <Graphic name="file-pkg" />
          <div className={`${baseClass}__info`}>
            <span className={`${baseClass}__title`}>
              {softwarePackage.name}
            </span>
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
      {showActions && (
        <div className={`${baseClass}__actions`}>
          <Button variant="icon" onClick={onAdvancedOptionsClick}>
            <Icon name="settings" color={"ui-fleet-black-75"} />
          </Button>
          {/* TODO: make a component for download icons */}
          <a
            className={`${baseClass}__download-icon`}
            href={downloadUrl}
            download
          >
            <Icon name="download" />
          </a>
          <Button variant="icon" onClick={onDeleteClick}>
            <Icon name="trash" color={"ui-fleet-black-75"} />
          </Button>
        </div>
      )}
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
        />
      )}
    </Card>
  );
};

export default SoftwarePackageCard;
