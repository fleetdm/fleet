import React, { useContext, useState } from "react";
import classnames from "classnames";

import { AppContext } from "context/app";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import { IInputFieldParseTarget } from "interfaces/form_field";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import {
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
} from "pages/SoftwarePage/helpers";

import generateFormValidation from "./helpers";
import { IAppStoreApp } from "interfaces/software";

const baseClass = "software-android-form";

export interface ISoftwareAndroidFormData {
  selfService: boolean;
  automaticInstall: boolean;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  applicationID: string;
  categories: string[];
  platform: "android";
}

export interface IFormValidation {
  isValid: boolean;
}

interface ISoftwareAndroidFormProps {
  softwareAndroidForEdit?: IAppStoreApp; // 4.77 Currently no edit Android functionality
  onSubmit: (formData: ISoftwareAndroidFormData) => void;
  isLoading?: boolean;
  onCancel: () => void;
  onClickPreviewEndUserExperience: () => void;
}

const SoftwareAndroidForm = ({
  softwareAndroidForEdit,
  onSubmit,
  isLoading = false,
  onCancel,
  onClickPreviewEndUserExperience,
}: ISoftwareAndroidFormProps) => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [formData, setFormData] = useState<ISoftwareAndroidFormData>(
    softwareAndroidForEdit
      ? {
          applicationID: softwareAndroidForEdit.app_store_id || "",
          selfService: softwareAndroidForEdit.self_service || false, // 4.77 Currently unavailable to change
          automaticInstall: softwareAndroidForEdit.automatic_install || false, // 4.77 Currently unavailable for Android apps
          targetType: getTargetType(softwareAndroidForEdit),
          customTarget: getCustomTarget(softwareAndroidForEdit),
          labelTargets: generateSelectedLabels(softwareAndroidForEdit),
          categories: softwareAndroidForEdit.categories || [],
          platform: "android",
        }
      : {
          applicationID: "",
          selfService: true, // Default to true for new Android apps
          automaticInstall: false, // 4.77 Currently navailable for Android apps
          targetType: "All hosts",
          customTarget: "labelsIncludeAny",
          labelTargets: {},
          categories: [],
          platform: "android",
        }
  );

  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: !!softwareAndroidForEdit, // Disables submit before Android application ID is entered
  });

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    setFormValidation(generateFormValidation(newFormData));
  };

  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
    setFormData(newData);
  };

  const onSelectCategory = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    let newCategories: string[];

    if (value) {
      // Add the name if not already present
      newCategories = formData.categories.includes(name)
        ? formData.categories
        : [...formData.categories, name];
    } else {
      // Remove the name if present
      newCategories = formData.categories.filter((cat) => cat !== name);
    }

    const newData = {
      ...formData,
      categories: newCategories,
    };

    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onToggleAutomaticInstall = (value: boolean) => {
    const newData = { ...formData, automaticInstall: value };
    setFormData(newData);
  };

  const isSubmitDisabled = !formValidation.isValid;

  const renderContent = () => {
    // Edit Android does not exist on 4.77
    if (softwareAndroidForEdit) {
      return null;
    }

    // Add Android form
    return (
      <div className={`${baseClass}__form-fields`}>
        <InputField
          autoFocus
          label="Application ID"
          placeholder="com.android.chrome"
          helpText={
            <>
              The ID at the end of the app&apos;s{" "}
              <CustomLink
                text="Google Play URL"
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/google-play-store`}
                newTab
              />{" "}
              E.g. &quot;com.android.chrome&quot; from
              &quot;https://play.google.com/store/apps/details?id=com.android.chrome&quot;
            </>
          }
          onChange={onInputChange}
          name="applicationID"
          value={formData.applicationID}
          parseTarget
          disabled={gitOpsModeEnabled} // TODO: Confirm GitOps behavior
        />
        <div className={`${baseClass}__form-frame`}>
          <Card paddingSize="medium" borderRadiusSize="large">
            <SoftwareOptionsSelector
              platform="android"
              formData={formData}
              onToggleAutomaticInstall={onToggleAutomaticInstall}
              onToggleSelfService={onToggleSelfServiceCheckbox}
              onSelectCategory={onSelectCategory}
              onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
              disableOptions
            />
          </Card>
        </div>
      </div>
    );
  };

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-disabled`]: isLoading,
  });

  const formContentClasses = classnames(`${baseClass}__form-content`, {
    [`${baseClass}__form-content--disabled`]: gitOpsModeEnabled,
  });

  return (
    <form className={baseClass} onSubmit={onFormSubmit}>
      {isLoading && <div className={`${baseClass}__overlay`} />}
      <div className={contentWrapperClasses}>
        <div className={formContentClasses}>
          <>{renderContent()}</>
        </div>
        <div className={`${baseClass}__action-buttons`}>
          <GitOpsModeTooltipWrapper
            position="bottom"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={disableChildren || isSubmitDisabled}
                isLoading={isLoading}
                className={`${baseClass}__add-software-btn`}
              >
                {softwareAndroidForEdit ? "Save" : "Add software"}
              </Button>
            )}
          />
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </form>
  );
};

export default SoftwareAndroidForm;
