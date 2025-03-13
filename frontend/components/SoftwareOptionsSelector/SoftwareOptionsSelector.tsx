import React from "react";
import classnames from "classnames";

import Checkbox from "components/forms/fields/Checkbox";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { SELF_SERVICE_TOOLTIP } from "pages/SoftwarePage/helpers";
import { ISoftwareVppFormData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareAppStoreVpp/SoftwareVppForm/SoftwareVppForm";
import { IFleetMaintainedAppFormData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/FleetAppDetailsForm";
import { IPackageFormData } from "pages/SoftwarePage/components/PackageForm/PackageForm";

const baseClass = "software-options-selector";

interface ISoftwareOptionsSelector {
  formData:
    | IFleetMaintainedAppFormData
    | ISoftwareVppFormData
    | IPackageFormData;
  /** Only used in create mode not edit mode for FMA, VPP, and custom packages */
  onToggleAutomaticInstall: (value: boolean) => void;
  onToggleSelfService: (value: boolean) => void;
  platform?: string;
  className?: string;
  isCustomPackage?: boolean;
  /** Edit mode does not have ability to change automatic install */
  isEditingSoftware?: boolean;
}

const SoftwareOptionsSelector = ({
  formData,
  onToggleAutomaticInstall,
  onToggleSelfService,
  platform,
  className,
  isCustomPackage,
  isEditingSoftware,
}: ISoftwareOptionsSelector) => {
  const classNames = classnames(baseClass, className);

  const isSelfServiceDisabled = platform === "ios" || platform === "ipados";
  const isAutomaticInstallDisabled =
    platform === "ios" || platform === "ipados";

  return (
    <div className="form-field">
      <div className="form-field__label">Options</div>
      {isSelfServiceDisabled && (
        <p>
          Currently, self-service and automatic installation are not available
          for iOS and iPadOS. Manually install on the <b>Host details</b> page
          for each host.
        </p>
      )}
      <Checkbox
        value={formData.selfService}
        onChange={(newVal: boolean) => onToggleSelfService(newVal)}
        className={`${baseClass}__self-service-checkbox`}
        tooltipContent={!isSelfServiceDisabled && SELF_SERVICE_TOOLTIP}
        disabled={isSelfServiceDisabled}
      >
        Self-service
      </Checkbox>
      {!isEditingSoftware && (
        <Checkbox
          value={formData.automaticInstall}
          onChange={(newVal: boolean) => onToggleAutomaticInstall(newVal)}
          className={`${baseClass}__automatic-install-checkbox`}
          tooltipContent={
            !isAutomaticInstallDisabled && (
              <>Automatically install only on hosts missing this software.</>
            )
          }
          disabled={isAutomaticInstallDisabled}
        >
          Automatic install
        </Checkbox>
      )}
      {formData.automaticInstall && isCustomPackage && (
        <InfoBanner color="yellow">
          Installing software over existing installations might cause issues.
          Fleet&apos;s policy may not detect these existing installations.
          Please create a test team in Fleet to verify a smooth installation.{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/query-templates-for-automatic-software-install`}
            text="Learn more"
            newTab
          />
        </InfoBanner>
      )}
    </div>
  );
};

export default SoftwareOptionsSelector;
