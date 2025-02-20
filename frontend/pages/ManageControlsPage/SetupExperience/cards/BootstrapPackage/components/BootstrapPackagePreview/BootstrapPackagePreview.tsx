import React from "react";

import OsSetupPreview from "../../../../../../../../assets/images/os-setup-preview.gif";

const baseClass = "bootstrap-package-preview";

const BootstrapPackagePreview = () => {
  return (
    <div className={baseClass}>
      <h3>End user experience</h3>
      <p>
        The bootstrap package is automatically installed after the end user
        authenticates and agrees to the EULA during the <b>Remote Management</b>{" "}
        screen in macOS Setup Assistant.
      </p>
      <p>
        The end user is allowed to continue to the next setup screen before the
        installation starts.
      </p>
      <p>The package isn&apos;t installed on hosts that already enrolled.</p>
      <img
        className={`${baseClass}__preview-img`}
        src={OsSetupPreview}
        alt="End user experience during the macOS setup assistant with the
        bootstrap package installation"
      />
    </div>
  );
};

export default BootstrapPackagePreview;
