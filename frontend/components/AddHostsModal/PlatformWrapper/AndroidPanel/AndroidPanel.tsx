import React, { useContext } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { getPathWithQueryParams } from "utilities/url";

import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";

import InputField from "components/forms/fields/InputField";

import EnrollQrCode from "../EnrollQrCode";

type EnrollmentType = "workProfile" | "fullyManaged";

const generateUrl = (
  serverUrl: string,
  enrollSecret: string,
  enrollType: EnrollmentType
) => {
  return getPathWithQueryParams(`${serverUrl}/enroll`, {
    enroll_secret: enrollSecret,
    platform: "android",
    fully_managed: enrollType === "fullyManaged" ? "true" : undefined,
  });
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
        <h3 className="platform-wrapper__panel-heading">
          Enrollment instructions
        </h3>
        <InputField
          label="Share this link with your end users:"
          enableCopy
          readOnly
          inputWrapperClass={`${baseClass}__enroll-link`}
          name="enroll-link"
          value={url}
        />
        <EnrollQrCode url={url} />
      </form>
    </div>
  );
};

export default AndroidPanel;
