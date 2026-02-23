import React, { useContext } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";

// @ts-ignore
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

  if (!config) return null;

  if (!isAndroidMdmEnabledAndConfigured) {
    return (
      <p>
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_MDM_ANDROID}
          text="Turn on Android MDM"
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
            label="Work profile"
            value="workProfile"
            checked={enrollmentType === "workProfile"}
            onChange={() => setEnrollmentType("workProfile")}
          />
          <Radio
            name="enrollmentType"
            id="fullyManaged"
            label="Fully-managed (no work profile)"
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
        />
      </form>
    </div>
  );
};

export default AndroidPanel;
