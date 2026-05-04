import React from "react";
import classnames from "classnames";

import Checkbox from "components/forms/fields/Checkbox";
import Slider from "components/forms/fields/Slider";
import CustomLink from "components/CustomLink";

import paths from "router/paths";
import { getSelfServiceTooltip } from "pages/SoftwarePage/helpers";
import { ISoftwareVppFormData } from "pages/SoftwarePage/components/forms/SoftwareVppForm/SoftwareVppForm";
import { IFleetMaintainedAppFormData } from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/FleetAppDetailsForm";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";
import { ISoftwareAndroidFormData } from "pages/SoftwarePage/components/forms/SoftwareAndroidForm/SoftwareAndroidForm";
import {
  CATEGORIES_ITEMS,
  ICategory,
} from "pages/hosts/details/cards/Software/SelfService/helpers";
import Button from "components/buttons/Button";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

const baseClass = "software-options-selector";

interface ICategoriesSelector {
  onSelectCategory: ({ name, value }: { name: string; value: boolean }) => void;
  selectedCategories: string[];
  onClickPreviewEndUserExperience: () => void;
}

export const AndroidOptionsDescription = () => (
  <p>
    Currently, Android apps can only be added as self-service and the end user
    can install them from the <strong>Play Store</strong> in their work profile.
    Additionally, you can install it when hosts enroll on the{" "}
    <CustomLink
      url={paths.CONTROLS_INSTALL_SOFTWARE("android")}
      text="Setup experience"
    />{" "}
    page.
  </p>
);

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
              >
                <div className={`${baseClass}__label-name`}>{cat.label}</div>
              </Checkbox>
            </div>
          );
        })}
      </div>
      <Button
        variant="inverse"
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
    | IPackageFormData
    | ISoftwareAndroidFormData;
  onToggleSelfService: () => void;
  onClickPreviewEndUserExperience: () => void;
  onSelectCategory: ({ name, value }: { name: string; value: boolean }) => void;
  platform?: string;
  className?: string;
  /** IPA packages do not have ability to select self-service */
  isIpaPackage?: boolean;
  isEditingSoftware?: boolean;
  disableOptions?: boolean;
}

const SoftwareOptionsSelector = ({
  formData,
  onToggleSelfService,
  onClickPreviewEndUserExperience,
  onSelectCategory,
  platform,
  className,
  isIpaPackage,
  isEditingSoftware,
  disableOptions = false,
}: ISoftwareOptionsSelector) => {
  const classNames = classnames(baseClass, className);

  const isPlatformIosOrIpados =
    isIPadOrIPhone(platform || "") || isIpaPackage || false;
  const isPlatformAndroid = isAndroid(platform || "");
  const isSelfServiceDisabled = disableOptions;

  // Ability to set categories when adding software is in a future ticket #28061
  const canSelectSoftwareCategories = formData.selfService && isEditingSoftware;

  const renderOptionsDescription = () =>
    isPlatformAndroid ? <AndroidOptionsDescription /> : null;

  const selfServiceLabelTooltip = !isSelfServiceDisabled
    ? getSelfServiceTooltip(isPlatformIosOrIpados, isPlatformAndroid)
    : undefined;

  return (
    <div className={`form-field ${classNames}`}>
      <div className="form-field__label">Options</div>
      {renderOptionsDescription()}
      <div className={`${baseClass}__self-service`}>
        <Slider
          value={formData.selfService}
          onChange={onToggleSelfService}
          inactiveText="Self-service"
          activeText="Self-service"
          labelTooltip={selfServiceLabelTooltip}
          className={`${baseClass}__self-service-slider`}
          disabled={isSelfServiceDisabled}
        />

        {canSelectSoftwareCategories && (
          <CategoriesSelector
            onSelectCategory={onSelectCategory}
            selectedCategories={formData.categories}
            onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          />
        )}
      </div>
    </div>
  );
};

export default SoftwareOptionsSelector;
