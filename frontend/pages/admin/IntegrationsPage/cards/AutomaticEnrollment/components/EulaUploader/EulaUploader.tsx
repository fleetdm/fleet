import React from "react";
import { noop } from "lodash";

import FileUploader from "pages/ManageControlsPage/components/FileUploader/FileUploader";

const baseClass = "eula-uploader";

interface IEulaUploaderProps {}

const EulaUploader = ({}: IEulaUploaderProps) => {
  return (
    <div className={baseClass}>
      <FileUploader
        icon="file-pdf"
        message="PDF (.pdf)"
        onFileUpload={noop}
        accept=".pdf"
      />
    </div>
  );
};

export default EulaUploader;
