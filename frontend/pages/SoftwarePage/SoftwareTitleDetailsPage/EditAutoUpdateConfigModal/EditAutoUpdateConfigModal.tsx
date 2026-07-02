import React, { useState } from "react";
import classnames from "classnames";
import { ISoftwareTitleDetails, IAppStoreApp } from "interfaces/software";

import useGitOpsMode from "hooks/useGitOpsMode";

import softwareAPI from "services/entities/software";

import { notify } from "components/ToastNotification";
import Card from "components/Card";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import {
  generateSelectedLabels,
  getCustomTarget,
  getDisplayedSoftwareName,
  getTargetType,
} from "pages/SoftwarePage/helpers";

import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

import {
  ISoftwareAutoUpdateConfigFormValidation,
  ISoftwareAutoUpdateConfigInputValidation,
  validateFormData,
} from "./helpers";

const baseClass = "edit-auto-update-config-modal";
const formClass = "edit-auto-update-config-form";

const getReadonlyTargetContent = (
  appStoreApp: IAppStoreApp | null
): JSX.Element => {
  if (appStoreApp?.labels_include_any?.length) {
    return (
      <>
        <p className={`${formClass}__target-description`}>
          Update settings will only apply to hosts that <b>have any</b> of these
          labels:
        </p>
        <ul className={`${formClass}__target-labels-list`}>
          {appStoreApp.labels_include_any.map((label) => (
            <li key={label.id}>{label.name}</li>
          ))}
        </ul>
      </>
    );
  }
  if (appStoreApp?.labels_include_all?.length) {
    return (
      <>
        <p className={`${formClass}__target-description`}>
          Update settings will only apply to hosts that <b>have all</b> of these
          labels:
        </p>
        <ul className={`${formClass}__target-labels-list`}>
          {appStoreApp.labels_include_all.map((label) => (
            <li key={label.id}>{label.name}</li>
          ))}
        </ul>
      </>
    );
  }
  if (appStoreApp?.labels_exclude_any?.length) {
    return (
      <>
        <p className={`${formClass}__target-description`}>
          Update settings will only apply to hosts that{" "}
          <b>don&apos;t have any</b> of these labels:
        </p>
        <ul className={`${formClass}__target-labels-list`}>
          {appStoreApp.labels_exclude_any.map((label) => (
            <li key={label.id}>{label.name}</li>
          ))}
        </ul>
      </>
    );
  }
  return (
    <p className={`${formClass}__target-description`}>
      Update settings will apply to <b>all hosts</b>.
    </p>
  );
};

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
  const { gitOpsModeEnabled } = useGitOpsMode("software");

  const clsNames = classnames(formClass, {
    [`edit-auto-update-config-form--disabled`]: gitOpsModeEnabled,
  });

  const [isUpdatingConfiguration, setIsUpdatingConfiguration] = useState(false);
  const [formData, setFormData] = useState<ISoftwareAutoUpdateConfigFormData>({
    autoUpdateEnabled: softwareTitle.auto_update_enabled || false,
    autoUpdateStartTime: softwareTitle.auto_update_window_start || "",
    autoUpdateEndTime: softwareTitle.auto_update_window_end || "",
    targetType: getTargetType(softwareTitle.app_store_app as IAppStoreApp),
    customTarget: getCustomTarget(softwareTitle.app_store_app as IAppStoreApp),
    labelTargets: generateSelectedLabels(
      softwareTitle.app_store_app as IAppStoreApp
    ),
  });

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

      notify.success(
        <>
          <strong>
            {getDisplayedSoftwareName(
              softwareTitle.name,
              softwareTitle.display_name
            )}
          </strong>{" "}
          configuration updated.
        </>
      );

      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      notify.error(
        "An error occurred while updating the configuration. Please try again.",
        { response: e }
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
      <div className={clsNames}>
        <div className={`${formClass}__form-frame`}>
          <Card paddingSize="medium" borderRadiusSize="medium">
            <div className={`${formClass}__auto-update-config`}>
              <div className={`form-field`}>
                <div className="form-field__label">Auto updates</div>
                <div className="form-field__subtitle">
                  Automatically update{" "}
                  <strong>
                    {getDisplayedSoftwareName(
                      softwareTitle.name,
                      softwareTitle.display_name
                    )}
                  </strong>{" "}
                  on all targeted hosts when a new version is available.
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
            <div className={`${formClass}__target-readonly`}>
              <div className="form-field__label">Target</div>
              {getReadonlyTargetContent(softwareTitle.app_store_app)}
              <p className={`${formClass}__target-edit-hint`}>
                To edit the target, close this modal and select{" "}
                <b>Actions &gt; Edit software</b>.
              </p>
            </div>
          </Card>
        </div>
      </div>
      <ModalFooter
        primaryButtons={
          <>
            <Button onClick={onExit} variant="inverse">
              Cancel
            </Button>
            <GitOpsModeTooltipWrapper
              entityType="software"
              position="top"
              tipOffset={8}
              renderChildren={(disableChildren) => (
                <Button
                  type="submit"
                  onClick={onSubmitForm}
                  isLoading={isUpdatingConfiguration}
                  disabled={
                    !formValidation.isValid ||
                    isUpdatingConfiguration ||
                    disableChildren
                  }
                >
                  Save
                </Button>
              )}
            />
          </>
        }
      />
    </Modal>
  );
};

export default EditAutoUpdateConfigModal;
