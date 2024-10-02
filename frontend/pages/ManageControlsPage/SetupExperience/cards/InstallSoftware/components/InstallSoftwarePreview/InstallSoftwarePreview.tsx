import React from "react";

import Card from "components/Card";

import InstallSoftwarePreviewImg from "../../../../../../../../assets/images/install-software-preview.png";

const baseClass = "install-software-preview";

const InstallSoftwarePreview = () => {
  return (
    <Card color="gray" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        When the end user completes the macOS Setup Assistant, they will see
        software being installed. User will not be able to continue until
        software completes installation.
      </p>
      <p>
        If there are any installation errors, the end user will be able to
        continue and will be instructed to contact their IT department.
      </p>
      <img
        className={`${baseClass}__preview-img`}
        src={InstallSoftwarePreviewImg}
        alt="install software preview"
      />
    </Card>
  );
};

export default InstallSoftwarePreview;
