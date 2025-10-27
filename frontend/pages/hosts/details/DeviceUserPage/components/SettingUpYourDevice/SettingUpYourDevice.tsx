import Card from "components/Card";
import { ISetupStep } from "interfaces/setup";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";
import Textarea from "components/Textarea";
import React, { useState } from "react";
import InfoButton from "../InfoButton";
import SetupStatusTable from "./SetupStatusTable";

import {
  hasRemainingSetupSteps,
  getFailedSoftwareInstall,
} from "../../helpers";

const baseClass = "setting-up-your-device";

interface ISettingUpYourDevice {
  setupSteps: ISetupStep[];
  toggleInfoModal: () => void;
  requireAllSoftware: boolean;
}

const SettingUpYourDevice = ({
  setupSteps,
  toggleInfoModal,
  requireAllSoftware,
}: ISettingUpYourDevice) => {
  const [showError, setShowError] = useState(false);
  let title;
  let message;
  const failedSoftware = requireAllSoftware
    ? getFailedSoftwareInstall(setupSteps)
    : null;
  if (failedSoftware) {
    title = "Device setup failed";
    message = (
      <>
        <p>
          Your organization requires that critical software be installed before
          you use your device. <b>{failedSoftware.name}</b> failed to install.
        </p>
        <p>
          <Icon name="error-outline" color="status-error" size="small" />{" "}
          <TooltipWrapper
            tipContent={<>CONTROL (⌃) + Command (⌘) + ⏻ or Touch ID</>}
          >
            Restart your device
          </TooltipWrapper>{" "}
          to try again. If this keeps happening, please contact your IT admin.
        </p>
      </>
    );
  } else if (hasRemainingSetupSteps(setupSteps)) {
    title = "Setting up your device...";
    message = (
      <p>
        Your computer is currently being configured by your organization. Please
        don&rsquo;t attempt to restart or shut down the computer unless prompted
        to do so.
      </p>
    );
  } else {
    title = "Configuration complete";
    message = (
      <p>
        Your computer has been successfully configured. Setup will continue
        momentarily.
      </p>
    );
  }

  return (
    <div className={`${baseClass} main-content device-user`}>
      <Card borderRadiusSize="xxlarge" paddingSize="xlarge">
        <div className={`${baseClass}__header`}>
          <h2>{title}</h2>
          {!failedSoftware && <InfoButton onClick={toggleInfoModal} />}
        </div>
        {message}
        {!failedSoftware && <SetupStatusTable statuses={setupSteps} />}
        {failedSoftware && (
          <div className={`${baseClass}__failure-state`}>
            <RevealButton
              className={`${baseClass}__accordion-title`}
              isShowing={showError}
              showText="Details"
              hideText="Details"
              caretPosition="after"
              onClick={() => setShowError(!showError)}
            />
            {showError && (
              <Textarea variant="code">{failedSoftware.error}</Textarea>
            )}
          </div>
        )}
      </Card>
    </div>
  );
};

export default SettingUpYourDevice;
