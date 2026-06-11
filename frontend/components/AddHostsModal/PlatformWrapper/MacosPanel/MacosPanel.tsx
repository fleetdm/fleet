import React, { useContext } from "react";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import InputField from "components/forms/fields/InputField";

type EnrollmentMethod = "enrollmentProfile" | "agentPackage";
type DeviceType = "companyOwned" | "personalBYOD";

const generateUrl = (
  serverUrl: string,
  enrollSecret: string,
  deviceType: DeviceType
) => {
  const url = `${serverUrl}/enroll?enroll_secret=${encodeURIComponent(
    enrollSecret
  )}`;
  if (deviceType === "personalBYOD") {
    return `${url}&byo=true`;
  }
  return url;
};

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

  const [
    enrollmentMethod,
    setEnrollmentMethod,
  ] = React.useState<EnrollmentMethod>(
    isMacMdmEnabledAndConfigured ? "enrollmentProfile" : "agentPackage"
  );
  const [deviceType, setDeviceType] = React.useState<DeviceType>(
    "companyOwned"
  );

  if (!config) return null;

  const enrollUrl = generateUrl(
    config.server_settings.server_url,
    enrollSecret,
    deviceType
  );

  const installerString = generateInstallerString(
    config.server_settings.server_url,
    enrollSecret,
    config.server_settings.scripts_disabled
  );

  return (
    <div className={baseClass}>
      <form>
        <div className="form-field">
          <div className="form-field__label">Enrollment method</div>
          <Radio
            className={`${baseClass}__radio-input`}
            label="Enrollment profile"
            id="enrollment-profile"
            checked={enrollmentMethod === "enrollmentProfile"}
            value="enrollmentProfile"
            name="enrollment-method"
            onChange={() => setEnrollmentMethod("enrollmentProfile")}
          />
          <Radio
            className={`${baseClass}__radio-input`}
            label="Agent package"
            id="agent-package"
            checked={enrollmentMethod === "agentPackage"}
            value="agentPackage"
            name="enrollment-method"
            onChange={() => setEnrollmentMethod("agentPackage")}
          />
        </div>
        {enrollmentMethod === "enrollmentProfile" &&
          (!isMacMdmEnabledAndConfigured ? (
            <p>
              <CustomLink
                url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
                text="Turn on Apple MDM"
              />{" "}
              to use the enrollment profile.
            </p>
          ) : (
            <>
              <div className="form-field">
                <div className="form-field__label">Device type</div>
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Personal (BYOD)"
                  id="personal-byod"
                  checked={deviceType === "personalBYOD"}
                  value="personalBYOD"
                  name="device-type"
                  onChange={() => setDeviceType("personalBYOD")}
                />
                <Radio
                  className={`${baseClass}__radio-input`}
                  label="Company-owned"
                  id="company-owned"
                  checked={deviceType === "companyOwned"}
                  value="companyOwned"
                  name="device-type"
                  onChange={() => setDeviceType("companyOwned")}
                />
              </div>
              <InputField
                readOnly
                inputWrapperClass={`${baseClass}__enroll-link`}
                name="enroll-link"
                enableCopy
                label="Enrollment instructions:"
                value={enrollUrl}
                helpText="When the end user navigates to this URL, the enrollment profile will download in their browser. End users will have to install the profile to enroll to Fleet. The Fleet agent will automatically be installed after the host is enrolled."
              />
            </>
          ))}
        {enrollmentMethod === "agentPackage" && (
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
            helpText="Run this on your computer, then deploy the generated package to your hosts. After installing the package, end users will have to install the MDM profile from Fleet Desktop to turn on MDM."
          />
        )}
      </form>
    </div>
  );
};

export default MacosPanel;
