import React, { useContext, useState } from "react";
import classnames from "classnames";
import { ISoftwareTitleDetails, IAppStoreApp } from "interfaces/software";
import { ILabelSummary } from "interfaces/label";

import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import softwareAPI from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import Card from "components/Card";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Checkbox from "components/forms/fields/Checkbox";
import TargetLabelSelector from "components/TargetLabelSelector";

import {
  CUSTOM_TARGET_OPTIONS,
  generateSelectedLabels,
  getCustomTarget,
  generateHelpText,
  getTargetType,
} from "pages/SoftwarePage/helpers";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import {
  ISoftwareAutoUpdateConfigFormValidation,
  ISoftwareAutoUpdateConfigInputValidation,
  validateFormData,
} from "./helpers";

const baseClass = "edit-auto-update-config-modal";
const formClass = "edit-auto-update-config-form";

// Schema for the form data that will be used in the UI
// and sent to the API.
export interface ISoftwareAutoUpdateConfigFormData {
  autoUpdateEnabled: boolean;
  autoUpdateStartTime: string;
  autoUpdateEndTime: string;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
}

interface EditAutoUpdateConfigModal {
  teamId: number;
  softwareTitle: ISoftwareTitleDetails;
  refetchSoftwareTitle: () => void;
  onExit: () => void;
}

const EditAutoUpdateConfigModal = ({
  softwareTitle,
  teamId,
  refetchSoftwareTitle,
  onExit,
}: EditAutoUpdateConfigModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled || false;

  const formClassNames = classnames(formClass, {
    [`edit-auto-update-config-form--disabled`]: gitOpsModeEnabled,
  });

  const [isUpdatingConfiguration, setIsUpdatingConfiguration] = useState(false);
  const [formData, setFormData] = useState<ISoftwareAutoUpdateConfigFormData>({
    autoUpdateEnabled: softwareTitle.auto_update_enabled || false,
    autoUpdateStartTime: softwareTitle.auto_update_start_time || "",
    autoUpdateEndTime: softwareTitle.auto_update_end_time || "",
    targetType: getTargetType(softwareTitle.app_store_app as IAppStoreApp),
    customTarget: getCustomTarget(softwareTitle.app_store_app as IAppStoreApp),
    labelTargets: generateSelectedLabels(
      softwareTitle.app_store_app as IAppStoreApp
    ),
  });

  // Fetch labels for TargetLabelSelector
  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const [
    formValidation,
    setFormValidation,
  ] = useState<ISoftwareAutoUpdateConfigFormValidation>(() =>
    validateFormData(formData)
  );

  // Currently calls the "edit app store app" API.
  // FUTURE: switch endpoint based on software title type?
  const onSubmitForm = async (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const newValidation = validateFormData(formData, true);
    setFormValidation(newValidation);

    if (!newValidation.isValid) {
      return false;
    }

    setIsUpdatingConfiguration(true);

    try {
      await softwareAPI.editAppStoreApp(softwareTitle.id, teamId, formData);

      renderFlash(
        "success",
        <>
          <strong>{softwareTitle.name}</strong> configuration updated.
        </>
      );

      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash(
        "error",
        "An error occurred while updating the configuration. Please try again."
      );
    }
    setIsUpdatingConfiguration(false);
    return true;
  };

  const onToggleEnabled = (value: boolean) => {
    const newFormData = { ...formData, autoUpdateEnabled: value };
    setFormData(newFormData);
    setFormValidation(validateFormData(newFormData));
  };

  const onChangeTimeField = (update: { name: string; value: string }) => {
    // Ensure HH:MM format with proper characters.
    const value = update.value.substring(0, 5).replace(/[^0-9:]/g, "");
    const newFormData = { ...formData, [update.name]: value };
    setFormData(newFormData);
    const newValidation = validateFormData(newFormData);
    // Can be "autoUpdateStartTime" or "autoUpdateEndTime".
    const fieldName = update.name as keyof ISoftwareAutoUpdateConfigFormValidation;
    const fieldValidation = newValidation[
      fieldName
    ] as ISoftwareAutoUpdateConfigInputValidation;
    // We don't want to show an error message as the user types.
    // (that will happen on blur instead)
    // We'll just clear any existing error if the field is valid.
    if (fieldValidation?.isValid) {
      setFormValidation(newValidation);
    }
  };

  const onSelectTargetType = (value: string) => {
    const newData = { ...formData, targetType: value };
    setFormData(newData);
    setFormValidation(validateFormData(newData));
  };

  const onSelectCustomTargetOption = (value: string) => {
    const newData = { ...formData, customTarget: value };
    setFormData(newData);
    setFormValidation(validateFormData(newData));
  };

  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    const newData = {
      ...formData,
      labelTargets: { ...formData.labelTargets, [name]: value },
    };
    setFormData(newData);
    setFormValidation(validateFormData(newData));
  };

  const earliestStartTimeError =
    formValidation.autoUpdateStartTime?.message ||
    (formValidation.windowLength?.message ? "Earliest start time" : undefined);

  const latestStartTimeError =
    formValidation.autoUpdateEndTime?.message ||
    (formValidation.windowLength?.message ? "Latest start time" : undefined);

  const updateWindowLabel = formValidation.windowLength?.message || (
    <>Update window (host&rsquo;s local time)</>
  );
  const updateWindowLabelClass = classnames("form-field__label", {
    "form-field__label--error": !!formValidation.windowLength?.message,
  });

  return (
    <Modal className={baseClass} title="Schedule auto updates" onExit={onExit}>
      <div className={formClassNames}>
        <div className={`${formClass}__form-frame`}>
          <Card paddingSize="medium" borderRadiusSize="medium">
            <div className={`${formClass}__auto-update-config`}>
              <div className={`form-field`}>
                <div className="form-field__label">Auto updates</div>
                <div className="form-field__subtitle">
                  Automatically update <strong>{softwareTitle.name}</strong> on
                  all targeted hosts when a new version is available.
                </div>
                <div>
                  <Checkbox
                    value={formData.autoUpdateEnabled}
                    onChange={(newVal: boolean) => onToggleEnabled(newVal)}
                  >
                    Enable auto updates
                  </Checkbox>
                </div>
              </div>
              {formData.autoUpdateEnabled && (
                <>
                  <div>
                    <div className="form-field">
                      <div className={updateWindowLabelClass}>
                        {updateWindowLabel}
                      </div>
                      <div className="form-field__subtitle">
                        Times are formatted as HH:MM in 24 hour time (e.g.,
                        &quot;13:37&quot;).
                      </div>
                    </div>
                  </div>
                  <div>
                    <div className={`${formClass}__auto-update-schedule-form`}>
                      <span className="date-time-inputs">
                        <InputField
                          value={formData.autoUpdateStartTime}
                          onChange={onChangeTimeField}
                          onBlur={() =>
                            setFormValidation(validateFormData(formData))
                          }
                          label="Earliest start time"
                          name="autoUpdateStartTime"
                          parseTarget
                          error={earliestStartTimeError}
                        />
                        <InputField
                          value={formData.autoUpdateEndTime}
                          onChange={onChangeTimeField}
                          onBlur={() =>
                            setFormValidation(validateFormData(formData))
                          }
                          label="Latest start time"
                          name="autoUpdateEndTime"
                          parseTarget
                          error={latestStartTimeError}
                        />
                      </span>
                    </div>
                  </div>
                </>
              )}
            </div>
          </Card>
          <Card paddingSize="medium" borderRadiusSize="medium">
            <TargetLabelSelector
              selectedTargetType={formData.targetType}
              selectedCustomTarget={formData.customTarget}
              selectedLabels={formData.labelTargets}
              customTargetOptions={CUSTOM_TARGET_OPTIONS}
              className={`${formClass}__target`}
              onSelectTargetType={onSelectTargetType}
              onSelectCustomTarget={onSelectCustomTargetOption}
              onSelectLabel={onSelectLabel}
              labels={labels || []}
              dropdownHelpText={
                generateHelpText(false, formData.customTarget) // maps to !automaticInstall help text
              }
              subTitle="Changes to targets will also apply to self-service."
            />
          </Card>
        </div>
        <ModalFooter
          primaryButtons={
            <>
              <Button onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button
                type="submit"
                onClick={onSubmitForm}
                isLoading={isUpdatingConfiguration}
                disabled={!formValidation.isValid || isUpdatingConfiguration}
              >
                Save
              </Button>
            </>
          }
        />
      </div>
    </Modal>
  );
};

export default EditAutoUpdateConfigModal;
