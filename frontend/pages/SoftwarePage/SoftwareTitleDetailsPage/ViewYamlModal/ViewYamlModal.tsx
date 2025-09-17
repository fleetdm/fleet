import React, { useContext } from "react";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getExtensionFromFileName } from "utilities/file/fileUtils";
import FileSaver from "file-saver";
import { ISoftwarePackage } from "interfaces/software";
import softwareAPI from "services/entities/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Editor from "components/Editor";

import { hyphenateString } from "utilities/strings/stringUtils";
import { createPackageYaml, renderDownloadFilesText } from "./helpers";

const baseClass = "view-yaml-modal";

interface IViewYamlModalProps {
  softwareTitleName: string;
  softwareTitleId: number;
  teamId: number;
  iconUrl?: string | null;
  softwarePackage: ISoftwarePackage;
  onExit: () => void;
}

interface HandleDownloadParams {
  evt: React.MouseEvent;
  content?: string;
  downloadUrl?: string;
  filename: string;
  filetype: string;
  errorMsg: string;
}

const ViewYamlModal = ({
  softwareTitleName,
  softwareTitleId: softwareId,
  teamId,
  iconUrl,
  softwarePackage,
  onExit,
}: IViewYamlModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);
  const repositoryUrl = config?.gitops?.repository_url;
  const {
    name,
    version,
    url,
    hash_sha256: sha256,
    pre_install_query: preInstallQuery,
    install_script: installScript,
    post_install_script: postInstallScript,
    uninstall_script: uninstallScript,
  } = softwarePackage;

  const packageYaml = createPackageYaml({
    softwareTitle: softwareTitleName,
    packageName: name,
    version,
    url,
    sha256,
    preInstallQuery,
    installScript,
    postInstallScript,
    uninstallScript,
    iconUrl: iconUrl || null,
  });

  // Generic download handler
  const handleDownload = async ({
    evt,
    content,
    downloadUrl,
    filename,
    filetype,
    errorMsg,
  }: HandleDownloadParams) => {
    evt.preventDefault();

    try {
      if (content) {
        const file = new window.File([content], filename, { type: filetype });
        FileSaver.saveAs(file);
      } else if (downloadUrl) {
        const response = await fetch(downloadUrl);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const blob = await response.blob();
        FileSaver.saveAs(blob, filename);
      } else {
        throw new Error("No content or URL provided");
      }
    } catch (err) {
      renderFlash("error", errorMsg);
    }
    return false;
  };

  const hyphenatedSoftwareTitle = hyphenateString(softwareTitleName);

  const onDownloadPreInstallQuery = (evt: React.MouseEvent) => {
    const softwareExtension = getExtensionFromFileName(name);
    const preInstallQueryContent = `- name: "[Pre-install software] ${softwareTitleName} (${softwareExtension})"\n  query: ${preInstallQuery}`;

    handleDownload({
      evt,
      content: preInstallQueryContent,
      filename: `pre-install-query-${hyphenatedSoftwareTitle}.yml`,
      filetype: "text/yml",
      errorMsg:
        "Your pre-install query could not be downloaded. Please create YAML file (.yml) manually.",
    });
  };

  const onDownloadPostInstallScript = (evt: React.MouseEvent) => {
    handleDownload({
      evt,
      content: postInstallScript,
      filename: `post-install-${hyphenatedSoftwareTitle}.sh`,
      filetype: "text/sh",
      errorMsg:
        "Your post-install script could not be downloaded. Please create script file (.sh) manually.",
    });
  };

  const onDownloadInstallScript = (evt: React.MouseEvent) => {
    handleDownload({
      evt,
      content: installScript,
      filename: `install-${hyphenatedSoftwareTitle}.sh`,
      filetype: "text/sh",
      errorMsg:
        "Your install script could not be downloaded. Please create script file (.sh) manually.",
    });
  };

  const onDownloadUninstallScript = (evt: React.MouseEvent) => {
    handleDownload({
      evt,
      content: uninstallScript,
      filename: `uninstall-${hyphenatedSoftwareTitle}.sh`,
      filetype: "text/sh",
      errorMsg:
        "Your uninstall script could not be downloaded. Please create script file (.sh) manually.",
    });
  };

  const onDownloadIcon = async (evt: React.MouseEvent) => {
    evt.preventDefault();

    try {
      // Get icon blob + create filename
      const response = await softwareAPI.getSoftwareIcon(softwareId, teamId);
      // Different from icon's original filename as we are suggesting a standard name used in YAML
      const filename = `${hyphenatedSoftwareTitle}-icon.png`;

      // Save the file
      FileSaver.saveAs(response.data, filename);
    } catch (err) {
      renderFlash(
        "error",
        "Your icon could not be downloaded. Please download the image manually."
      );
    }
  };

  return (
    <Modal className={baseClass} title="YAML" onExit={onExit}>
      <>
        <InfoBanner className={`${baseClass}__info-banner`}>
          <p>
            To complete your GitOps configuration, follow the instructions
            below. If the YAML is not added, new installers will be deleted on
            the next GitOps run, and edited installers will cause the GitOps run
            to fail.&nbsp;
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/yaml-packages`}
              text="How to use YAML"
              newTab
              multiline
              variant="banner-link"
            />
          </p>
        </InfoBanner>
        {repositoryUrl && (
          <p>
            First, create the YAML file below and save it to your{" "}
            <CustomLink url={repositoryUrl} text="repository" newTab />.
          </p>
        )}
        <p>Make sure you reference the package YAML from your team YAML.</p>
        <div className={`${baseClass}__form-fields`}>
          <InputField
            enableCopy
            readOnly
            name="filename"
            label="Filename"
            value={`${hyphenatedSoftwareTitle}.package.yml`}
          />
          <Editor label="Contents" value={packageYaml} enableCopy />
        </div>
        <p>
          {renderDownloadFilesText({
            preInstallQuery,
            installScript,
            postInstallScript,
            uninstallScript,
            iconUrl,
            onClickPreInstallQuery: preInstallQuery
              ? onDownloadPreInstallQuery
              : undefined,
            onClickInstallScript: installScript
              ? onDownloadInstallScript
              : undefined,
            onClickPostInstallScript: postInstallScript
              ? onDownloadPostInstallScript
              : undefined,
            onClickUninstallScript: uninstallScript
              ? onDownloadUninstallScript
              : undefined,
            onClickIcon: iconUrl ? onDownloadIcon : undefined,
          })}
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ViewYamlModal;
