import React from "react";
import classnames from "classnames";

import Checkbox from "components/forms/fields/Checkbox";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { SELF_SERVICE_TOOLTIP } from "pages/SoftwarePage/helpers";
import { ISoftwareVppFormData } from "pages/SoftwarePage/components/forms/SoftwareVppForm/SoftwareVppForm";
import { IFleetMaintainedAppFormData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/FleetAppDetailsForm";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";
import {
  CATEGORIES_ITEMS,
  ICategory,
} from "pages/hosts/details/cards/Software/SelfService/helpers";
import Button from "components/buttons/Button";

const baseClass = "software-options-selector";

interface ICategoriesSelector {
  onSelectCategory: ({ name, value }: { name: string; value: boolean }) => void;
  selectedCategories: any; // TODO
  onClickPreviewEndUserExperience: () => void;
}

const CategoriesSelector = ({
  onSelectCategory,
  selectedCategories,
  onClickPreviewEndUserExperience,
}: ICategoriesSelector) => {
  return (
    <>
      <div className="form-field__label">Categories</div>
      <div className={`${baseClass}__categories-selector`}>
        {CATEGORIES_ITEMS.map((cat: ICategory) => {
          return (
            <div className={`${baseClass}__label`} key={cat.id}>
              <Checkbox
                className={`${baseClass}__checkbox`}
                name={cat.value}
                value={selectedCategories.includes(cat.value)}
                onChange={onSelectCategory}
                parseTarget
              />
              <div className={`${baseClass}__label-name`}>{cat.label}</div>
            </div>
          );
        })}
      </div>
      <Button
        variant="text-link"
        onClick={onClickPreviewEndUserExperience}
        className={`${baseClass}__preview-button`}
      >
        Preview end user experience
      </Button>
    </>
  );
};

interface ISoftwareOptionsSelector {
  formData:
    | IFleetMaintainedAppFormData
    | ISoftwareVppFormData
    | IPackageFormData;
  /** Only used in create mode not edit mode for FMA, VPP, and custom packages */
  onToggleAutomaticInstall: (value: boolean) => void;
  onToggleSelfService: (value: boolean) => void;
  onClickPreviewEndUserExperience: () => void;
  onSelectCategory: ({ name, value }: { name: string; value: boolean }) => void;
  platform?: string;
  className?: string;
  isCustomPackage?: boolean;
  /** Exe packages do not have ability to select automatic install */
  isExePackage?: boolean;
  /** Tarball packages do not have ability to select automatic install */
  isTarballPackage?: boolean;
  /** Edit mode does not have ability to change automatic install */
  isEditingSoftware?: boolean;
  disableOptions?: boolean;
}

const SoftwareOptionsSelector = ({
  formData,
  onToggleAutomaticInstall,
  onToggleSelfService,
  onClickPreviewEndUserExperience,
  onSelectCategory,
  platform,
  className,
  isCustomPackage,
  isExePackage,
  isTarballPackage,
  isEditingSoftware,
  disableOptions = false,
}: ISoftwareOptionsSelector) => {
  const classNames = classnames(baseClass, className);

  const isPlatformIosOrIpados = platform === "ios" || platform === "ipados";
  const isSelfServiceDisabled = disableOptions || isPlatformIosOrIpados;
  const isAutomaticInstallDisabled =
    disableOptions || isPlatformIosOrIpados || isExePackage || isTarballPackage;

  /** Tooltip only shows when enabled or for exe/tar.gz packages */
  const showAutomaticInstallTooltip =
    !isAutomaticInstallDisabled || isExePackage || isTarballPackage;
  const getAutomaticInstallTooltip = (): JSX.Element => {
    if (isExePackage || isTarballPackage) {
      return (
        <>
          Fleet can&apos;t create a policy to detect existing installations for{" "}
          {isExePackage ? ".exe packages" : ".tar.gz archives"}. To
          automatically install{" "}
          {isExePackage ? ".exe packages" : ".tar.gz archives"}, add a custom
          policy and enable the install software automation on the{" "}
          <b>Policies</b> page.
        </>
      );
    }
    return <>Automatically install only on hosts missing this software.</>;
  };

  // Ability to set categories when adding software is in a future ticket #28061
  const canSelectSoftwareCategories = formData.selfService && isEditingSoftware;

  return (
    <div className={`form-field ${classNames}`}>
      <div className="form-field__label">Options</div>
      {isPlatformIosOrIpados && (
        <p>
          Currently, self-service and automatic installation are not available
          for iOS and iPadOS. Manually install on the <b>Host details</b> page
          for each host.
        </p>
      )}
      <div className={`${baseClass}__self-service`}>
        <Checkbox
          value={formData.selfService}
          onChange={(newVal: boolean) => onToggleSelfService(newVal)}
          className={`${baseClass}__self-service-checkbox`}
          labelTooltipContent={!isSelfServiceDisabled && SELF_SERVICE_TOOLTIP}
          disabled={isSelfServiceDisabled}
        >
          Self-service
        </Checkbox>
        {canSelectSoftwareCategories && (
          <CategoriesSelector
            onSelectCategory={onSelectCategory}
            selectedCategories={formData.categories}
            onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          />
        )}
      </div>
      {!isEditingSoftware && (
        <Checkbox
          value={formData.automaticInstall}
          onChange={(newVal: boolean) => onToggleAutomaticInstall(newVal)}
          className={`${baseClass}__automatic-install-checkbox`}
          labelTooltipContent={
            showAutomaticInstallTooltip && getAutomaticInstallTooltip()
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
