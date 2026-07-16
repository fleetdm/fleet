import React from "react";

import InfoBanner from "components/InfoBanner/InfoBanner";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import Button from "components/buttons/Button";

const baseClass = "api-key-display";

interface IApiKeyDisplayProps {
  newUserName: string;
  apiKey: string;
  onDone: () => void;
}

const ApiKeyDisplay = ({
  newUserName,
  apiKey,
  onDone,
}: IApiKeyDisplayProps) => {
  return (
    <>
      <h1>{newUserName}</h1>
      <div className={baseClass}>
        <div className={`${baseClass}__api-key-label`}>
          <b>API Key</b>
        </div>
        <InputFieldHiddenContent value={apiKey} name="api-key" />
        <InfoBanner color="yellow">
          Please make a note of this API key since it is the only time you will
          be able to view it.
        </InfoBanner>
        <div>
          <Button onClick={onDone}>Done</Button>
        </div>
      </div>
    </>
  );
};

export default ApiKeyDisplay;
