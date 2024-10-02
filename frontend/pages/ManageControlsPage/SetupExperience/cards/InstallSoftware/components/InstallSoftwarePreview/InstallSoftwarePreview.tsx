import React from "react";

import Card from "components/Card";

import OsPrefillPreview from "../../../../../../../../assets/images/os-prefill-preview.gif";

const baseClass = "install-software-preview";

const InstallSoftwarePreview = () => {
  return (
    <Card color="gray" paddingSize="xxlarge" className={baseClass}>
      <h2>End user experience</h2>
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
        src={OsPrefillPreview}
        alt="OS setup preview"
      />
    </Card>
  );
};

export default InstallSoftwarePreview;
