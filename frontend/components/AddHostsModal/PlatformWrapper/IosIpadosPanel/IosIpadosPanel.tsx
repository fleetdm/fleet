import React, { useContext } from "react";

import CustomLink from "components/CustomLink";
import PATHS from "router/paths";
import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";

const generateUrl = (serverUrl: string, enrollSecret: string) => {
  return `${serverUrl}/enroll?enroll_secret=${encodeURIComponent(
    enrollSecret
  )}`;
};

const baseClass = "ios-ipados-panel";

interface IosIpadosPanelProps {
  enrollSecret: string;
}

const IosIpadosPanel = ({ enrollSecret }: IosIpadosPanelProps) => {
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);

  const helpText =
    "When the end user navigates to this URL, the enrollment profile " +
    "will download in their browser. End users will have to install the profile " +
    "to enroll to Fleet.";

  if (!config) return null;

  if (!isMacMdmEnabledAndConfigured) {
    return (
      <p>
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
          text="Turn on Apple MDM"
        />{" "}
        to enroll iOS & iPadOS hosts.
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
        helpText={helpText}
      />
    </div>
  );
};

export default IosIpadosPanel;
