import React, { useContext, useState } from "react";
import { ISoftwareTitleDetails, IAppStoreApp } from "interfaces/software";
import { ILabelSummary } from "interfaces/label";

import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";

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

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary().then((res) => getCustomLabels(res.labels)),
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

  // Edit package API call
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
      renderFlash("error", "");
    }
    setIsUpdatingConfiguration(false);
    return true;
  };

  const onToggleEnabled = (value: boolean) => {
    const newFormData = { ...formData, autoUpdateEnabled: value };
    setFormData(newFormData);
    setFormValidation(validateFormData(newFormData));
    // setCanSaveForm(!error);
  };

  const onChangeTimeField = (update: { name: string; value: string }) => {
    const value = update.value.substring(0, 5).replace(/[^0-9:]/g, ""); // limit to 5 characters and allow only numbers and colon
    const newFormData = { ...formData, [update.name]: value };
    setFormData(newFormData);
    const newValidation = validateFormData(newFormData);
    const fieldName = update.name as keyof ISoftwareAutoUpdateConfigFormValidation;
    const fieldValidation = newValidation[
      fieldName
    ] as ISoftwareAutoUpdateConfigInputValidation;
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

  const renderForm = () => (
    <div className={`${baseClass}__form-frame`}>
      <Card paddingSize="medium" borderRadiusSize="medium">
        <div className={`${formClass}__auto-update-config`}>
          <div className={`form-field`}>
            <div className="form-field__label">Auto updates</div>
            <div className="form-field__subtitle">
              Automatically update <strong>{softwareTitle.name}</strong> on all
              targeted hosts when a new version is available.
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
                  <div className="form-field__label">
                    Update window (host&rsquo;s local time)
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
                      error={formValidation.autoUpdateStartTime?.message}
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
                      error={formValidation.autoUpdateEndTime?.message}
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
  );

  return (
    <Modal className={baseClass} title="Schedule auto updates" onExit={onExit}>
      <div className={formClass}>
        {renderForm()}
        <ModalFooter
          primaryButtons={
            <Button
              type="submit"
              onClick={onSubmitForm}
              isLoading={isUpdatingConfiguration}
              disabled={!formValidation.isValid || isUpdatingConfiguration}
            >
              Save
            </Button>
          }
        />
      </div>
    </Modal>
  );
};

export default EditAutoUpdateConfigModal;
