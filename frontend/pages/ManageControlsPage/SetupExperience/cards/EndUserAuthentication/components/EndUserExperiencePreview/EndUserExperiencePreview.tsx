import React from "react";
import classnames from "classnames";

import EndUserAuthPreviewVideo from "../../../../../../../../assets/videos/end-user-auth.mp4";

console.log(EndUserAuthPreviewVideo);

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
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        className={`${baseClass}__preview-video`}
        src={EndUserAuthPreviewVideo}
        controls
        autoPlay
        loop
        muted
      />
    </div>
  );
};

export default EndUserExperiencePreview;
