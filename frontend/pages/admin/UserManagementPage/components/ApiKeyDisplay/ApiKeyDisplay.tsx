import React from "react";

import InfoBanner from "components/InfoBanner/InfoBanner";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import Button from "components/buttons/Button";

const baseClass = "api-key-display";

interface IApiKeyDisplayProps {
  apiKey: string;
  onDone: () => void;
}

const ApiKeyDisplay = ({ apiKey, onDone }: IApiKeyDisplayProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__api-key-label`}>
        <b>API key</b>
      </div>
      <InputFieldHiddenContent value={apiKey} name="api-key" />
      <InfoBanner color="yellow">
        Please make a note of this API key since it is the only time you will be
        able to view it.
      </InfoBanner>
      <div className={`${baseClass}__done-button`}>
        <Button onClick={onDone}>Done</Button>
      </div>
    </div>
  );
};

export default ApiKeyDisplay;
