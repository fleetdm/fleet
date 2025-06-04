import React from "react";

import Card from "components/Card";

import InstallSoftwareEndUserPreview from "../../../../../../../../assets/videos/install-software-preview.mp4";

const baseClass = "install-software-preview";

const InstallSoftwarePreview = () => {
  return (
    <Card color="grey" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        During the <b>Remote Management</b> screen, the end user will see
        selected software being installed. They won&apos;t be able to continue
        until software is installed.
      </p>
      <p>
        If there are any errors, they will be able to continue and will be
        instructed to contact their IT admin.
      </p>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        className={`${baseClass}__preview-video`}
        src={InstallSoftwareEndUserPreview}
        controls
        autoPlay
        loop
        muted
      />
    </Card>
  );
};

export default InstallSoftwarePreview;
