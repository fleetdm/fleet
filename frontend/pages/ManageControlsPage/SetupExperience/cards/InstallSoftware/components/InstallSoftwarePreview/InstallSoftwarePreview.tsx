import React from "react";

import Card from "components/Card";

// TODO - update this video once full UI/Server/Agent/End user flow is integrated
import InstallSoftwareEndUserPreview from "../../../../../../../../assets/videos/install-software-preview.mp4";

const baseClass = "install-software-preview";

const InstallSoftwarePreview = () => {
  return (
    <Card color="grey" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        When Fleet&apos;s agent (fleetd) is installed, fleetd will open the{" "}
        <b>Fleet Desktop &gt; My device</b> page in the default browser.
      </p>
      <p>The end user will see selected software being installed.</p>
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
