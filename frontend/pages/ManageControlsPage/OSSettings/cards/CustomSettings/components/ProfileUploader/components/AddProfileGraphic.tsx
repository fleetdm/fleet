import React from "react";

import Graphic from "components/Graphic";

const ProfileGraphic = ({
  baseClass,
  showMessage,
}: {
  baseClass: string;
  showMessage?: boolean;
}) => (
  <div className={`${baseClass}__profile-graphic`}>
    <Graphic
      key="file-configuration-profile-graphic"
      className={`${baseClass}__graphic`}
      name="file-configuration-profile"
    />
    {showMessage && (
      <span className={`${baseClass}__profile-graphic--message`}>
        <b>Upload configuration profile</b>
        <br />
        .mobileconfig and .json for macOS, iOS, and iPadOS.
        <br />
        .xml for Windows.
      </span>
    )}
  </div>
);

export default ProfileGraphic;
