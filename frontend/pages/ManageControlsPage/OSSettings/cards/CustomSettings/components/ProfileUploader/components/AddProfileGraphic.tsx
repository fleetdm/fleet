import React from "react";

import Graphic from "components/Graphic";

const ALLOWED_FILE_TYPES_MESSAGE =
  "Configuration profile (.mobileconfig for macOS or .xml for Windows)";

const ProfileGraphic = ({
  baseClass,
  showMessage,
}: {
  baseClass: string;
  showMessage?: boolean;
}) => (
  <div className={`${baseClass}__profile-graphic`}>
    <Graphic
      key={`file-configuration-profile-graphic`}
      className={`${baseClass}__graphic`}
      name="file-configuration-profile"
    />
    {showMessage && (
      <span className={`${baseClass}__profile-graphic--message`}>
        {ALLOWED_FILE_TYPES_MESSAGE}
      </span>
    )}
  </div>
);

export default ProfileGraphic;
