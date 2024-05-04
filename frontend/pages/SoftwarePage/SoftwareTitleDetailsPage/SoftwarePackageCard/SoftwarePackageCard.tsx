import React from "react";

import { ISoftwarePackage } from "interfaces/software";

import Card from "components/Card";
import Graphic from "components/Graphic";
import { uploadedFromNow } from "utilities/date_format";
import TooltipWrapper from "components/TooltipWrapper";
import { internationalTimeFormat } from "utilities/helpers";

const baseClass = "software-package-card";

interface ISoftwarePackageCardProps {
  softwarePackage: ISoftwarePackage;
}

const SoftwarePackageCard = ({
  softwarePackage,
}: ISoftwarePackageCardProps) => {
  return (
    <Card className={baseClass}>
      <div className={`${baseClass}__main-content`}>
        <Graphic name="file-pkg" />
        <div className={`${baseClass}__info`}>
          <span className={`${baseClass}__title`}>{softwarePackage.name}</span>
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
      <div>status</div>
      <div className={`${baseClass}__actions`}>test</div>
    </Card>
  );
};

export default SoftwarePackageCard;
