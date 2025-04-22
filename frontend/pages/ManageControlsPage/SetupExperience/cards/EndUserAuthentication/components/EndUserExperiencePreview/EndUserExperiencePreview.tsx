import React from "react";
import classnames from "classnames";

import OsSetupPreview from "../../../../../../../../assets/images/os-setup-preview.gif";

const baseClass = "end-user-experience-preview";

interface IEndUserExperiencePreviewProps {
  className?: string;
}

const EndUserExperiencePreview = ({
  className,
}: IEndUserExperiencePreviewProps) => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <h3>End user experience</h3>
      <p>
        When the end user reaches the <b>Remote Management</b> screen, they are
        first asked to authenticate and agree to the end user license agreement
        (EULA).
      </p>
      <img
        className={`${baseClass}__preview-img`}
        src={OsSetupPreview}
        alt="End user experience during the macOS setup assistant with the user
        logging in with their IdP provider"
      />
    </div>
  );
};

export default EndUserExperiencePreview;
