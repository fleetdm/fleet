import React, { useContext, useState } from "react";

import CustomLink from "components/CustomLink";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import { getPathWithQueryParams } from "utilities/url";

import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";
import { renderAppleManualEnrollmentDisabled } from "components/AddHostsModal/helpers";

import EnrollQrCode from "../EnrollQrCode";

type EnrollmentType = "personal" | "companyOwned";

const baseClass = "ios-ipados-panel";

interface IosIpadosPanelProps {
  enrollSecret: string;
  isManualAppleEnrollmentsBlocked: boolean;
}

const IosIpadosPanel = ({
  enrollSecret,
  isManualAppleEnrollmentsBlocked,
}: IosIpadosPanelProps) => {
  const { config, isMacMdmEnabledAndConfigured } = useContext(AppContext);

  // Default to "Personal (BYOD)" per #23242 design.
  const [enrollmentType, setEnrollmentType] = useState<EnrollmentType>(
    "personal"
  );

  if (!config) return null;

  if (isManualAppleEnrollmentsBlocked) {
    return renderAppleManualEnrollmentDisabled("iOS & iPadOS");
  }

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

  // This tab covers both iOS and iPadOS, so the link can't know in advance
  // which of the two the end user will open it on — "ios" is just a hint
  // for enroll-ota.html to route into the combined iOS/iPadOS instructions
  // (and desktop QR handoff) instead of guessing from the previewing
  // browser's user agent. The enrolling device's own user agent still
  // decides iPhone- vs iPad-specific wording/images.
  const url = getPathWithQueryParams(
    `${config.server_settings.server_url}/enroll`,
    {
      enroll_secret: enrollSecret,
      byod: enrollmentType === "personal" ? "true" : undefined,
      platform: "ios",
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
