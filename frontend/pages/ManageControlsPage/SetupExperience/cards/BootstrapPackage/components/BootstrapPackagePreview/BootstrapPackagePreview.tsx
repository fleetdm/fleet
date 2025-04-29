import React from "react";

import BootstrapPackageEndUserPreview from "../../../../../../../../assets/videos/bootstrap-package-preview.mp4";

const baseClass = "bootstrap-package-preview";

const BootstrapPackagePreview = () => {
  return (
    <div className={baseClass}>
      <h3>End user experience</h3>
      <p>
        The bootstrap package is installed after the end user authenticates and
        agrees to the EULA.
      </p>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        className={`${baseClass}__preview-video`}
        src={BootstrapPackageEndUserPreview}
        controls
        autoPlay
        loop
        muted
      />
    </div>
  );
};

export default BootstrapPackagePreview;
