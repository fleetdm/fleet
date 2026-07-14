import React, { useContext } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";

import InputField from "components/forms/fields/InputField";

type EnrollmentType = "workProfile" | "fullyManaged";

const generateUrl = (
  serverUrl: string,
  enrollSecret: string,
  enrollType: EnrollmentType
) => {
  const url = `${serverUrl}/enroll?enroll_secret=${encodeURIComponent(
    enrollSecret
  )}`;

  if (enrollType === "fullyManaged") {
    return `${url}&fully_managed=true`;
  }

  return url;
};

const baseClass = "android-panel";

interface IAndroidPanelProps {
  enrollSecret: string;
}

const AndroidPanel = ({ enrollSecret }: IAndroidPanelProps) => {
  const { config, isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  const [enrollmentType, setEnrollmentType] = React.useState<EnrollmentType>(
    "workProfile"
  );

  const helpText =
    "When the end user navigates to this URL, the enrollment profile " +
    "will download in their browser. End users will have to install the profile " +
    "to enroll to Fleet.";

  if (!config) return null;

  if (!isAndroidMdmEnabledAndConfigured) {
    return (
      <p>
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_MDM_ANDROID}
          text="Turn on Android MDM"
          emphasized
        />{" "}
        to enroll Android hosts.
      </p>
    );
  }

  const url = generateUrl(
    config.server_settings.server_url,
    enrollSecret,
    enrollmentType
  );

  return (
    <div className={baseClass}>
      <form>
        <fieldset className="form-field">
          <Radio
            name="enrollmentType"
            id="workProfile"
            label="Personal (BYOD)"
            value="workProfile"
            checked={enrollmentType === "workProfile"}
            onChange={() => setEnrollmentType("workProfile")}
          />
          <Radio
            name="enrollmentType"
            id="fullyManaged"
            label="Company-owned (fully-managed)"
            value="fullyManaged"
            checked={enrollmentType === "fullyManaged"}
            onChange={() => setEnrollmentType("fullyManaged")}
          />
        </fieldset>
        <InputField
          label="Enrollment instructions:"
          enableCopy
          readOnly
          inputWrapperClass={`${baseClass}__enroll-link`}
          name="enroll-link"
          value={url}
          helpText={helpText}
        />
      </form>
    </div>
  );
};

export default AndroidPanel;
