import React, { useContext } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { AppContext } from "context/app";

const baseClass = "ios-ipados-panel";

const IosIpadosPanel = () => {
  const { config } = useContext(AppContext);
  console.log(config);

  const helpText =
    "When the end user navigates to this URL, the enrollment profile " +
    "will download in their browser. End users will have to install the profile " +
    "to enroll to Fleet.";

  return (
    <div className={baseClass}>
      <InputField
        label="Send this to your end users:"
        enableCopy
        readOnly
        inputWrapperClass
        name="enroll-link"
        value="https://example.com"
        helpText={helpText}
      />
    </div>
  );
};

export default IosIpadosPanel;
