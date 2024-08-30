import React from "react";

import { getFileDetails } from "utilities/file/fileUtils";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FleetAce from "components/FleetAce";
import CustomLink from "components/CustomLink";
import Editor from "components/Editor";
import {
  FileUploader,
  FileDetails,
} from "components/FileUploader/FileUploader";
import { noop } from "lodash";

const baseClass = "edit-software-modal";

interface IEditSoftwareModalProps {
  software: any; // TODO
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  onSubmit: () => void;
  onExit: () => void;
}

const EditSoftwareModal = ({
  software,
  installScript,
  preInstallQuery,
  postInstallScript,
  uninstallScript,
  onSubmit,
  onExit,
}: IEditSoftwareModalProps) => {
  return (
    <Modal className={baseClass} title="Edit software" onExit={onExit}>
      <>
        <div className={`${baseClass}__form-inputs`}>
          <FileUploader
            editFile
            graphicName={"file-pkg"}
            accept=".pkg,.msi,.exe,.deb"
            message=".pkg, .msi, .exe, or .deb"
            // onFileUpload={onFileUpload}
            onFileUpload={noop}
            buttonMessage="Choose file"
            buttonType="link"
            className={`${baseClass}__file-uploader`}
            filePreview={
              software && <FileDetails details={getFileDetails(software)} />
            }
          />
          <div className={`${baseClass}__input-field`}>
            <FleetAce
              value={preInstallQuery}
              label="Pre-install query"
              name="preInstallQuery"
              maxLines={10}
              helpText={
                <>
                  Software will be installed only if the{" "}
                  <CustomLink
                    className={`${baseClass}__table-link`}
                    text="query returns results"
                    url="https://fleetdm.com/tables"
                    newTab
                  />
                </>
              }
            />
          </div>
          <div className={`${baseClass}__input-field`}>
            <Editor
              label="Install script"
              wrapEnabled
              maxLines={10}
              name="install-script"
              value={installScript}
              helpText="Fleet will run this command on hosts to install software."
            />
          </div>
          <div className={`${baseClass}__input-field`}>
            <Editor
              label="Post-install script"
              wrapEnabled
              name="post-install-script-editor"
              maxLines={10}
              value={postInstallScript}
              helpText="Shell (macOS and Linux) or PowerShell (Windows)."
            />
          </div>
          <div className={`${baseClass}__input-field`}>
            <Editor
              label="Uninstall script"
              wrapEnabled
              name="post-install-script-editor"
              maxLines={10}
              value={uninstallScript}
              helpText="Shell (macOS and Linux) or PowerShell (Windows)."
            />
          </div>
        </div>
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onSubmit}>
            Save
          </Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default EditSoftwareModal;
