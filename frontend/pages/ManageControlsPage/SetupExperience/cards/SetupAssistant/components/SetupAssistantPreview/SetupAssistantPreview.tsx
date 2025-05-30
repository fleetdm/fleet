import React from "react";

import Card from "components/Card";

import SetupAssistantEndUserPreview from "../../../../../../../../assets/videos/setup-assistant-preview.mp4";

const baseClass = "setup-assistant-preview";

const SetupAssistantPreview = () => {
  return (
    <Card color="grey" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        After the end user continues past the <b>Remote Management</b> screen,
        macOS Setup Assistant displays several screens by default.
      </p>
      <p>
        By adding an automatic enrollment profile you can customize which
        screens are displayed and more.
      </p>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
      <video
        className={`${baseClass}__preview-video`}
        src={SetupAssistantEndUserPreview}
        controls
        autoPlay
        loop
        muted
      />
    </Card>
  );
};

export default SetupAssistantPreview;
