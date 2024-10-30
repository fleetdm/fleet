import React from "react";

import Card from "components/Card";

import OsPrefillPreview from "../../../../../../../../assets/images/os-prefill-preview.gif";

const baseClass = "setup-assistant-preview";

const SetupAssistantPreview = () => {
  return (
    <Card color="gray" paddingSize="xxlarge" className={baseClass}>
      <h3>End user experience</h3>
      <p>
        After the end user continues past the <b>Remote Management</b> screen,
        macOS Setup Assistant displays several screens by default.
      </p>
      <p>
        By adding an automatic enrollment profile you can customize which
        screens are displayed and more.
      </p>
      <img
        className={`${baseClass}__preview-img`}
        src={OsPrefillPreview}
        alt="End user experience during the macOS setup assistant customised by
        an automatic enrollment profile"
      />
    </Card>
  );
};

export default SetupAssistantPreview;
