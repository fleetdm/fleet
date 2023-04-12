import React, { useContext, useState } from "react";

import PATHS from "router/paths";

import Button from "components/buttons/Button/Button";
import { AppContext } from "context/app";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

const baseClass = "macos-setup";

interface ISetupEmptyState {
  router: any;
}

const SetupEmptyState = ({ router }: ISetupEmptyState) => {
  const onClickEmptyConnect = () => {
    router.push(PATHS.CONTROLS_MAC_SETTINGS);
  };

  return (
    <div className={`${baseClass}__empty-state`}>
      <h2>Setup experience for macOS hosts</h2>
      <p>Connect Fleet to the Apple Business Manager to get started.</p>
      <Button variant="brand" onClick={onClickEmptyConnect}>
        Connect
      </Button>
    </div>
  );
};

interface IMacOSSetupProps {
  router: any;
}

const MacOSSetup = ({ router }: IMacOSSetupProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const [isConfigured, setIsConfigured] = useState(false);

  if (isConfigured) return <SetupEmptyState router={router} />;

  return (
    <div className={baseClass}>
      <p>
        Customize the setup experience for hosts that automatically enroll to
        this team.
      </p>
      {isPremiumTier ? <PremiumFeatureMessage /> : <p>test</p>}
    </div>
  );
};

export default MacOSSetup;
