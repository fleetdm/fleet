import React, { useContext, useState } from "react";

import { AppContext } from "context/app";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import InputField from "components/forms/fields/InputField";

type DeviceType = "companyOwned" | "personalBYOD";

const generateInstallerString = (
  serverUrl: string,
  enrollSecret: string,
  scriptsDisabled: boolean
) => {
  return `fleetctl package --type=pkg ${
    !scriptsDisabled ? "--enable-scripts " : ""
  }--fleet-desktop --fleet-url=${serverUrl} --enroll-secret=${enrollSecret}`;
};

const baseClass = "macos-panel";

interface IMacosPanelProps {
  enrollSecret: string;
}

const MacosPanel = ({ enrollSecret }: IMacosPanelProps) => {
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);

  const [deviceType, setDeviceType] = useState<DeviceType>("companyOwned");

  if (!config) return null;

  if (isMacMdmEnabledAndConfigured) {
    const enrollUrl = getPathWithQueryParams(
      `${config.server_settings.server_url}/enroll`,
      {
        enroll_secret: enrollSecret,
        byod: deviceType === "personalBYOD" ? "true" : undefined,
      }
    );

    return (
      <div className={baseClass}>
        <form>
          <fieldset className="form-field">
            <Radio
              label="Personal (BYOD)"
              id="personal-byod"
              checked={deviceType === "personalBYOD"}
              value="personalBYOD"
              name="device-type"
              onChange={() => setDeviceType("personalBYOD")}
            />
            <Radio
              label="Company-owned"
              id="company-owned"
              checked={deviceType === "companyOwned"}
              value="companyOwned"
              name="device-type"
              onChange={() => setDeviceType("companyOwned")}
            />
          </fieldset>
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__enroll-link`}
            name="enroll-link"
            enableCopy
            label="Send this to your end users:"
            value={enrollUrl}
          />
        </form>
      </div>
    );
  }

  const installerString = generateInstallerString(
    config.server_settings.server_url,
    enrollSecret,
    config.server_settings.scripts_disabled
  );

  return (
    <div className={baseClass}>
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__installer-input`}
        name="installer"
        enableCopy
        label={
          <>
            Use this command to generate Fleet&apos;s agent.{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/generate-fleets-agent`}
              text="Learn how"
              newTab
            />
          </>
        }
        type="textarea"
        value={installerString}
        helpText="Run this on your computer, then deploy the generated package to your hosts."
      />
    </div>
  );
};

export default MacosPanel;
