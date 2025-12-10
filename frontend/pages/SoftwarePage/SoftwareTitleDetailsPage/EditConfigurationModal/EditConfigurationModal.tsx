import React, { useContext, useEffect, useState, useCallback } from "react";
import { IAppStoreApp, ISoftwarePackage } from "interfaces/software";
import { IInputFieldParseTarget } from "interfaces/form_field";

import { NotificationContext } from "context/notification";
import { INotification } from "interfaces/notification";
import { getErrorReason } from "interfaces/errors";
import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Editor from "components/Editor";
import Button from "components/buttons/Button";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import InstallerDetailsWidget from "../SoftwareInstallerCard/InstallerDetailsWidget";

const baseClass = "edit-configuration-modal";

// Used to surface error.message in UI of unknown error type
type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

const isErrorWithMessage = (error: unknown): error is ErrorWithMessage => {
  return (error as ErrorWithMessage).message !== undefined;
};

interface EditConfigurationModal {
  softwareInstaller: IAppStoreApp;
  onExit: () => void;
}

const EditConfigurationModal = ({
  softwareInstaller,
  onExit,
}: EditConfigurationModal) => {
  const { renderFlash, renderMultiFlash } = useContext(NotificationContext);

  const [isUpdatingConfiguration, setIsUpdatingConfiguration] = useState(false);
  const [canSaveForm, setCanSaveForm] = useState(true);
  const [jsonFormData, setJsonFormData] = useState<string>(
    JSON.stringify(softwareInstaller.configuration, null, "\t") || "{}"
  );
  const [formError, setFormError] = useState<string | null>(null);

  const validateForm = (curFormData: string) => {
    let error = null;

    if (curFormData) {
      try {
        JSON.parse(curFormData);
      } catch (e: unknown) {
        if (isErrorWithMessage(e)) {
          error = e.message.toString();
        } else {
          throw e;
        }
      }
    }
    return error;
  };

  const onInputChange = (value: string, event?: any) => {
    // TODO: handle input change
    setJsonFormData(value);
    setFormError(validateForm(value));
  };

  const renderHelpText = () => {
    return (
      <div className={`${baseClass}__help-text`}>
        The Android app&apos;s configuration in JSON format.{" "}
        <CustomLink
          newTab
          text="Learn more"
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/ui-gitops-mode`}
        />
      </div>
    );
  };

  const renderForm = () => (
    <>
      <Editor
        mode="json"
        value={jsonFormData as string}
        helpText={renderHelpText()}
        onChange={onInputChange}
        defaultValue="{}"
        error={formError}
        label="Configuration"
      />
    </>
  );

  const onClickSave = async () => {
    setIsUpdatingConfiguration(true);
  };

  return (
    <Modal className={baseClass} title="Edit configuration" onExit={onExit}>
      <>
        <InstallerDetailsWidget
          softwareName={softwareInstaller.name}
          androidPlayStoreId="com.android.chrome" // TODO: pass real value
          // androidPlayStoreId={softwareInstaller.app_store_id}
          customDetails="Android"
          installerType="app-store"
          isFma={false}
          isScriptPackage={false}
        />
        {renderForm()}
        <ModalFooter
          primaryButtons={
            <Button
              type="submit"
              onClick={onClickSave}
              isLoading={isUpdatingConfiguration}
              disabled={!canSaveForm || isUpdatingConfiguration}
            >
              Save
            </Button>
          }
        />
      </>
    </Modal>
  );
};

export default EditConfigurationModal;
