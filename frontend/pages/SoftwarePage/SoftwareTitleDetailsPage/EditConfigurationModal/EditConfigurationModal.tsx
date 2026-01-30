import React, { useContext, useState } from "react";
import { IAppStoreApp } from "interfaces/software";

import { NotificationContext } from "context/notification";

import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Editor from "components/Editor";
import Button from "components/buttons/Button";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import InstallerDetailsWidget from "../SoftwareInstallerCard/InstallerDetailsWidget";
import { getErrorMessage } from "./helpers";

const baseClass = "edit-configuration-modal";

// Used to surface error.message in UI of unknown error type
type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

const isErrorWithMessage = (error: unknown): error is ErrorWithMessage => {
  return (error as ErrorWithMessage).message !== undefined;
};

export interface ISoftwareConfigurationFormData {
  configuration: string;
}

interface EditConfigurationModal {
  softwareId: number;
  teamId: number;
  softwareInstaller: IAppStoreApp;
  refetchSoftwareTitle: () => void;
  onExit: () => void;
}

const EditConfigurationModal = ({
  softwareInstaller,
  softwareId,
  teamId,
  refetchSoftwareTitle,
  onExit,
}: EditConfigurationModal) => {
  const { renderFlash } = useContext(NotificationContext);

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

  // Edit package API call
  const onEditConfiguration = async (
    evt: React.MouseEvent<HTMLFormElement>
  ) => {
    setIsUpdatingConfiguration(true);

    evt.preventDefault();

    // Format for API
    const formDataToSubmit =
      jsonFormData === ""
        ? { configuration: {} } // Send empty object if no keys are set
        : {
            configuration: (jsonFormData && JSON.parse(jsonFormData)) || null,
          };
    try {
      await softwareAPI.editAppStoreApp(softwareId, teamId, formDataToSubmit);

      renderFlash(
        "success",
        <>
          <strong>{softwareInstaller.name}</strong> configuration updated.
        </>
      );

      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash(
        "error",
        getErrorMessage(e, softwareInstaller as IAppStoreApp)
      );
    }
    setIsUpdatingConfiguration(false);
  };

  const onInputChange = (value: string) => {
    setJsonFormData(value);

    const error = validateForm(value);
    setFormError(error);
    setCanSaveForm(!error);
  };

  const renderHelpText = () => {
    return (
      <div className={`${baseClass}__help-text`}>
        The Android app&apos;s configuration in JSON format.{" "}
        <CustomLink
          newTab
          text="Learn more"
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/android-software-managed-configuration`}
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
        error={formError}
        label="Configuration"
      />
    </>
  );

  return (
    <Modal className={baseClass} title="Edit configuration" onExit={onExit}>
      <>
        <InstallerDetailsWidget
          softwareName={softwareInstaller.name}
          androidPlayStoreId={softwareInstaller.app_store_id}
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
              onClick={onEditConfiguration}
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
