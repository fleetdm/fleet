import React, { useContext, useState } from "react";
import { ISoftwareTitleDetails } from "interfaces/software";

import { NotificationContext } from "context/notification";

import softwareAPI from "services/entities/software";

import Card from "components/Card";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getErrorMessage } from "./helpers";

const baseClass = "edit-auto-update-config-modal";
const formClass = "edit-auto-update-config-form";

// Used to surface error.message in UI of unknown error type
type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

const isErrorWithMessage = (error: unknown): error is ErrorWithMessage => {
  return (error as ErrorWithMessage).message !== undefined;
};

export interface ISoftwareAutoUpdateConfigFormData {
  enabled: boolean;
  startTime: string;
  endTime: string;
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
  const [canSaveForm, setCanSaveForm] = useState(true);
  const [formData, setFormData] = useState<ISoftwareAutoUpdateConfigFormData>({
    enabled: softwareTitle.auto_update_enabled || false,
    startTime: softwareTitle.auto_update_start_time || "",
    endTime: softwareTitle.auto_update_end_time || "",
  });

  const [formError, setFormError] = useState<string | null>(null);

  const validateForm = (curFormData: ISoftwareAutoUpdateConfigFormData) => {
    let error = null;
    error = null;
    return error;
  };

  // Edit package API call
  const onSubmitForm = async (evt: React.MouseEvent<HTMLFormElement>) => {
    setIsUpdatingConfiguration(true);

    evt.preventDefault();

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
        getErrorMessage(e, softwareTitle as ISoftwareTitleDetails)
      );
    }
    setIsUpdatingConfiguration(false);
  };

  const onToggleEnabled = (value: boolean) => {
    const newFormData = { ...formData, enabled: value };
    setFormData(newFormData);
    const error = validateForm(newFormData);
    setFormError(error);
    setCanSaveForm(!error);
  };

  const onChangeTimeField = (update: { name: string; value: string }) => {
    const value = update.value.substring(0, 5).replace(/[^0-9:]/g, ""); // limit to 5 characters and allow only numbers and colon
    const newFormData = { ...formData, [update.name]: value };
    setFormData(newFormData);
    const error = validateForm(newFormData);
    setFormError(error);
    setCanSaveForm(!error);
  };

  const renderForm = () => (
    <div className={`${baseClass}__form-frame`}>
      <Card paddingSize="medium" borderRadiusSize="medium">
        <div className={`${formClass}__auto-update-config`}>
          <div className={`form-field`}>
            <div className="form-field__label">Auto updates</div>
            <p>
              Automatically update <strong>{softwareTitle.name}</strong> on all
              targeted hosts when a new version is available.
            </p>
            <div>
              <Checkbox
                value={formData.enabled}
                onChange={(newVal: boolean) => onToggleEnabled(newVal)}
              >
                Enable auto updates
              </Checkbox>
            </div>
          </div>
          {formData.enabled && (
            <div>
              <div className="form-field">
                <div className="form-field__label">
                  Update window (host&rsquo;s local time)
                </div>
                <p>
                  Times are formatted as HH:MM in 24 hour time (e.g.,
                  &quot;13:37&quot;).
                </p>
              </div>
              <div className={`${formClass}__auto-update-schedule-form`}>
                <span className="date-time-inputs">
                  <InputField
                    value={formData.startTime}
                    onChange={onChangeTimeField}
                    label="Earliest start time"
                    name="startTime"
                    parseTarget
                  />
                  <InputField
                    value={formData.endTime}
                    onChange={onChangeTimeField}
                    label="Latest start time"
                    name="endTime"
                    parseTarget
                  />
                </span>
              </div>
            </div>
          )}
        </div>
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
              disabled={!canSaveForm || isUpdatingConfiguration}
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
