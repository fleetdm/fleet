import React from "react";

import Graphic from "components/Graphic";

interface IProfileGraphicProps {
  baseClass: string;
  /** Provide an optional message to be displayed below the graphic */
  message?: React.ReactNode;
}

const ProfileGraphic = ({ baseClass, message }: IProfileGraphicProps) => (
  <div className={`${baseClass}__profile-graphic`}>
    <Graphic
      key="file-configuration-profile-graphic"
      className={`${baseClass}__graphic`}
      name="file-configuration-profile"
    />
    {message && (
      <span className={`${baseClass}__profile-graphic--message`}>
        {message}
      </span>
    )}
  </div>
);

export default ProfileGraphic;
