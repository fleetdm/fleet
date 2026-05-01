import React, { useContext, useState } from "react";
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

  const getInitialValue = () => {
    if (isApplePlatform) {
      return softwareInstaller.configuration || "";
    }
    return JSON.stringify(softwareInstaller.configuration, null, "\t") || "{}";
  };

  const [isUpdatingConfiguration, setIsUpdatingConfiguration] = useState(false);
  const [canSaveForm, setCanSaveForm] = useState(true);
  const [formData, setFormData] = useState<string>(getInitialValue());
  const [formError, setFormError] = useState<string | null>(null);

  const validateForm = (curFormData: string): string | null => {
    if (isApplePlatform) {
      return validateXml(curFormData);
    }
    return validateJson(curFormData);
  };

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

  const platformLabel = getPlatformLabel(platform);
  const editorMode = isApplePlatform ? "xml" : "json";
  const formatLabel = isApplePlatform ? "XML" : "JSON";
  const learnMoreSlug = isApplePlatform
    ? "ios-ipados-software-managed-configuration"
    : "android-software-managed-configuration";

  const renderHelpText = () => {
    return (
      <div className={`${baseClass}__help-text`}>
        The {platformLabel} app&apos;s configuration in {formatLabel} format.{" "}
        <CustomLink
          newTab
          text="Learn more"
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/${learnMoreSlug}`}
        />
      </div>
    );
  };

  const renderForm = () => (
    <>
      <Editor
        mode={editorMode}
        value={formData}
        helpText={renderHelpText()}
        onChange={onInputChange}
        error={formError}
        label="Configuration"
      />
    </>
  );

  return (
    <Modal className={baseClass} title="Edit configuration" onExit={onExit}>
      <InstallerDetailsWidget
        softwareName={softwareInstaller.name}
        androidPlayStoreId={
          isApplePlatform ? undefined : softwareInstaller.app_store_id
        }
        customDetails={platformLabel}
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
    </Modal>
  );
};

export default EditConfigurationModal;
