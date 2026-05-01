import React, { useCallback, useContext, useState } from "react";
import { Ace } from "ace-builds";
import { IAppStoreApp } from "interfaces/software";
import { isIPadOrIPhone } from "interfaces/platform";

import { NotificationContext } from "context/notification";

import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Editor from "components/Editor";
import Button from "components/buttons/Button";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import InstallerDetailsWidget from "../SoftwareInstallerCard/InstallerDetailsWidget";
import {
  getErrorMessage,
  validateJson,
  validateXml,
  getPlatformLabel,
} from "./helpers";
import { getDisplayedSoftwareName } from "../../helpers";

const baseClass = "edit-configuration-modal";

export interface ISoftwareConfigurationFormData {
  configuration: string;
}

interface IEditConfigurationModalProps {
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
}: IEditConfigurationModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const platform = softwareInstaller.platform;
  const isApplePlatform = isIPadOrIPhone(platform);

  const XML_EMPTY = "<dict>\n  \n</dict>";

  const validateForm = (curFormData: string): string | null => {
    if (isApplePlatform) {
      return validateXml(curFormData);
    }
    return validateJson(curFormData);
  };

  const getInitialValue = () => {
    if (isApplePlatform) {
      return softwareInstaller.configuration || XML_EMPTY;
    }
    return JSON.stringify(softwareInstaller.configuration, null, "\t") || "{}";
  };

  // Place cursor between <dict> tags when starting with empty XML scaffold
  const isEmptyAppleConfig =
    isApplePlatform && !softwareInstaller.configuration;
  const onEditorLoad = useCallback(
    (editor: Ace.Editor) => {
      if (isEmptyAppleConfig) {
        // Row 1 (0-indexed) is the blank line between <dict> and </dict>
        editor.moveCursorTo(1, 2);
        editor.clearSelection();
      }
    },
    [isEmptyAppleConfig]
  );

  const initialValue = getInitialValue();

  const [isUpdatingConfiguration, setIsUpdatingConfiguration] = useState(false);
  const [canSaveForm, setCanSaveForm] = useState(!validateForm(initialValue));
  const [formData, setFormData] = useState<string>(initialValue);
  const [formError, setFormError] = useState<string | null>(null);

  const buildSubmitPayload = (): ISoftwareConfigurationFormData => {
    if (isApplePlatform) {
      // iOS/iPadOS: send XML as a string
      return { configuration: formData };
    }
    // Android: send parsed JSON object (cast to string to match interface;
    // runtime value is an object that gets serialized by sendRequest)
    if (formData === "") {
      return { configuration: ({} as unknown) as string };
    }
    return {
      configuration: (JSON.parse(formData) as unknown) as string,
    };
  };

  const onEditConfiguration = async (
    evt: React.MouseEvent<HTMLFormElement>
  ) => {
    setIsUpdatingConfiguration(true);

    evt.preventDefault();

    try {
      await softwareAPI.editAppStoreApp(
        softwareId,
        teamId,
        buildSubmitPayload()
      );

      renderFlash(
        "success",
        <>
          <strong>
            {getDisplayedSoftwareName(
              softwareInstaller.name,
              softwareInstaller.display_name
            )}
          </strong>{" "}
          configuration updated.
        </>
      );

      refetchSoftwareTitle();
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e, isApplePlatform));
    }
    setIsUpdatingConfiguration(false);
  };

  const onInputChange = (value: string) => {
    setFormData(value);

    const error = validateForm(value);
    setFormError(error);
    setCanSaveForm(!error);
  };

  const editorMode = isApplePlatform ? "xml" : "json";

  const renderHelpText = () => {
    if (isApplePlatform) {
      return (
        <div className={`${baseClass}__help-text`}>
          Managed app configuration, also known as App Config.{" "}
          <CustomLink
            newTab
            text="Learn more"
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/ios-ipados-software-managed-configuration`}
          />
        </div>
      );
    }
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

  const renderDescription = () => {
    if (!isApplePlatform) {
      return null;
    }
    return (
      <p className={`${baseClass}__description`}>
        Configuration and updated values of Fleet{" "}
        <CustomLink
          newTab
          text="variables"
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/fleet-variables`}
        />{" "}
        will be applied to future installs and updates.
      </p>
    );
  };

  const renderForm = () => (
    <Editor
      mode={editorMode}
      value={formData}
      helpText={renderHelpText()}
      onChange={onInputChange}
      onLoad={onEditorLoad}
      error={formError}
      label="Configuration"
    />
  );

  const renderInstallerDetails = () => {
    if (isApplePlatform) {
      return (
        <InstallerDetailsWidget
          softwareName={softwareInstaller.name}
          installerType="app-store"
          version={softwareInstaller.latest_version}
          isFma={false}
          isScriptPackage={false}
        />
      );
    }
    return (
      <InstallerDetailsWidget
        softwareName={softwareInstaller.name}
        androidPlayStoreId={softwareInstaller.app_store_id}
        customDetails={getPlatformLabel(platform)}
        installerType="app-store"
        isFma={false}
        isScriptPackage={false}
      />
    );
  };

  const renderFooter = () => {
    if (isApplePlatform) {
      return (
        <ModalFooter
          primaryButtons={
            <>
              <Button onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button
                type="submit"
                onClick={onEditConfiguration}
                isLoading={isUpdatingConfiguration}
                disabled={!canSaveForm || isUpdatingConfiguration}
              >
                Save
              </Button>
            </>
          }
        />
      );
    }
    return (
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
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Edit configuration"
      onExit={onExit}
      width="large"
    >
      {renderInstallerDetails()}
      {renderDescription()}
      {renderForm()}
      {renderFooter()}
    </Modal>
  );
};

export default EditConfigurationModal;
