import React, { useContext, useState } from "react";

import CustomLink from "components/CustomLink";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import { getPathWithQueryParams } from "utilities/url";

import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";

import EnrollQrCode from "../EnrollQrCode";

type EnrollmentType = "personal" | "companyOwned";

const baseClass = "ios-ipados-panel";

interface IosIpadosPanelProps {
  enrollSecret: string;
}

const IosIpadosPanel = ({ enrollSecret }: IosIpadosPanelProps) => {
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);

  // Default to "Personal (BYOD)" per #23242 design.
  const [enrollmentType, setEnrollmentType] = useState<EnrollmentType>(
    "personal"
  );

  if (!config) return null;

  if (!isMacMdmEnabledAndConfigured) {
    return (
      <p>
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_MDM_APPLE}
          text="Turn on Apple MDM"
          emphasized
        />{" "}
        to enroll iOS & iPadOS hosts.
      </p>
    );
  }

  const url = getPathWithQueryParams(
    `${config.server_settings.server_url}/enroll`,
    {
      enroll_secret: enrollSecret,
      byod: enrollmentType === "personal" ? "true" : undefined,
    }
  );

  return (
    <div className={baseClass}>
      <form>
        <fieldset className="form-field">
          <Radio
            name="iosIpadosEnrollmentType"
            id="iosIpadosPersonal"
            label="Personal (BYOD)"
            value="personal"
            checked={enrollmentType === "personal"}
            onChange={() => setEnrollmentType("personal")}
          />
          <Radio
            name="iosIpadosEnrollmentType"
            id="iosIpadosCompanyOwned"
            label="Company-owned (fully-managed)"
            value="companyOwned"
            checked={enrollmentType === "companyOwned"}
            onChange={() => setEnrollmentType("companyOwned")}
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

export default IosIpadosPanel;
