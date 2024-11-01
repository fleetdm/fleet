import React from "react";

import Card from "components/Card";

import InstallSoftwarePreviewImg from "../../../../../../../../assets/images/install-software-preview.png";

const baseClass = "install-software-preview";

const InstallSoftwarePreview = () => {
  return (
    <Card color="gray" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        After the <b>Remote Management</b> screen, the end user will see
        software being installed. They will not be able to continue until
        software is installed.
      </p>
      <p>
        If there are any errors, they will be able to continue and will be
        instructed to contact their IT admin.
      </p>
      <img
        className={`${baseClass}__preview-img`}
        src={InstallSoftwarePreviewImg}
        alt="End user experience during the macOS setup assistant with selected
        software being installed"
      />
    </Card>
  );
};

export default InstallSoftwarePreview;
