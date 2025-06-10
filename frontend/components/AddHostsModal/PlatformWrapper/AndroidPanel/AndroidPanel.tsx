import React, { useContext } from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";

const generateUrl = (serverUrl: string, enrollSecret: string) => {
  return `${serverUrl}/enroll?enroll_secret=${encodeURIComponent(
    enrollSecret
  )}`;
};

const baseClass = "android-panel";

interface IAndroidPanelProps {
  enrollSecret: string;
}

const AndroidPanel = ({ enrollSecret }: IAndroidPanelProps) => {
  const { config, isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  if (!config) return null;

  if (!isAndroidMdmEnabledAndConfigured) {
    return (
      <p>
        <Link to={PATHS.ADMIN_INTEGRATIONS_MDM_ANDROID}>
          Turn on Android MDM
        </Link>{" "}
        to enroll Android hosts.
      </p>
    );
  }

  const url = generateUrl(config.server_settings.server_url, enrollSecret);

  return (
    <div className={baseClass}>
      <InputField
        label="Send this to your end users:"
        enableCopy
        readOnly
        inputWrapperClass
        name="enroll-link"
        value={url}
      />
    </div>
  );
};

export default AndroidPanel;
