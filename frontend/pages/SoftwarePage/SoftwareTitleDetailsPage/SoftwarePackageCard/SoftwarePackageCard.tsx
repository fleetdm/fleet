import React from "react";

import { ISoftwarePackage } from "interfaces/software";

import Card from "components/Card";
import Graphic from "components/Graphic";
import { uploadedFromNow } from "utilities/date_format";
import TooltipWrapper from "components/TooltipWrapper";
import { internationalTimeFormat } from "utilities/helpers";
import DataSet from "components/DataSet";
import Icon from "components/Icon";

const baseClass = "software-package-card";

type IPackageInstallStatus = "installed" | "pending" | "failed";
interface IStatusDisplayOption {
  displayName: string;
  iconName: "success" | "pending-outline" | "error";
  tooltip: string;
}

const STATUS_DISPLAY_OPTIONS: Record<
  IPackageInstallStatus,
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
  status: IPackageInstallStatus;
  count: number;
}

const PackageStatusCount = ({ status, count }: IPackageStatusCountProps) => {
  const displayData = STATUS_DISPLAY_OPTIONS[status];
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
      value={<a className={`${baseClass}__status-count`}>{count} hosts</a>}
    />
  );
};

interface ISoftwarePackageCardProps {
  softwarePackage: ISoftwarePackage;
}

const SoftwarePackageCard = ({
  softwarePackage,
}: ISoftwarePackageCardProps) => {
  return (
    <Card className={baseClass}>
      <div className={`${baseClass}__main-content`}>
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
            status="installed"
            count={softwarePackage.status.installed}
          />
          <PackageStatusCount
            status="pending"
            count={softwarePackage.status.pending}
          />
          <PackageStatusCount
            status="failed"
            count={softwarePackage.status.failed}
          />
        </div>
      </div>
      <div className={`${baseClass}__actions`}>test</div>
    </Card>
  );
};

export default SoftwarePackageCard;
