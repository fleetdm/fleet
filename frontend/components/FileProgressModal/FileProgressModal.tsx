import Card from "components/Card";
import FileDetails from "components/FileDetails";
import Modal from "components/Modal";
import { noop } from "lodash";
import React from "react";

import { ISupportedGraphicNames } from "components/FileUploader/FileUploader";

import { IFileDetails } from "utilities/file/fileUtils";

const baseClass = "file-progress-modal";

const FileProgressModal = ({
  graphicNames = "file-pkg",
  fileDetails,
  fileProgress,
}: {
  graphicNames?: ISupportedGraphicNames | ISupportedGraphicNames[];
  fileDetails: IFileDetails;
  fileProgress: number;
}) => (
  <Modal
    className={baseClass}
    title="Add software"
    width="large"
    onExit={noop}
    disableClosingModal
  >
    <Card color="gray" className={`${baseClass}__card`}>
      <FileDetails
        graphicNames={graphicNames}
        fileDetails={fileDetails}
        progress={fileProgress}
        canEdit={false}
        onFileSelect={noop}
      />
    </Card>
  </Modal>
);

export default FileProgressModal;
