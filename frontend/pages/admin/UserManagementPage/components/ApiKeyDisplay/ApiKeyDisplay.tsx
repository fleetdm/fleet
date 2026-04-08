import React from "react";

import InfoBanner from "components/InfoBanner/InfoBanner";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import Button from "components/buttons/Button";

const baseClass = "api-key-display";

interface IApiKeyDisplayProps {
  apiKey: string;
  userName: string;
  onDone: () => void;
}

const ApiKeyDisplay = ({ apiKey, userName, onDone }: IApiKeyDisplayProps) => {
  return (
    <div className={baseClass}>
      <p className={`${baseClass}__success-message`}>
        <b>{userName}</b> has been created.
      </p>
      <InputFieldHiddenContent value={apiKey} name="api-key" />
      <InfoBanner>
        Copy this API key now. You won&apos;t be able to see it again. If you
        lose it, you&apos;ll need to create a new API-only user.
      </InfoBanner>
      <div className={`${baseClass}__done-button`}>
        <Button onClick={onDone}>Done</Button>
      </div>
    </div>
  );
};

export default ApiKeyDisplay;
