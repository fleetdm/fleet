import React, { useContext } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import CustomLink from "components/CustomLink/CustomLink";
import InfoBanner from "components/InfoBanner";
import Icon from "components/Icon";
import PageDescription from "components/PageDescription";
import Card from "components/Card";

const generateMdmTermsOfUseUrl = (domain: string) => {
  return `${domain}/api/mdm/microsoft/tos`;
};

const generateMdmDiscoveryUrl = (domain: string) => {
  return `${domain}/api/mdm/microsoft/discovery`;
};

const baseClass = "windows-automatic-enrollment-page";

const WindowsAutomaticEnrollmentPage = () => {
  const { config } = useContext(AppContext);

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text="Back to MDM"
            path={PATHS.ADMIN_INTEGRATIONS_MDM}
            className={`${baseClass}__back-to-automatic-enrollment`}
          />
        </div>
        <h1>Microsoft Entra</h1>
        <PageDescription
          content={
            <>
              To connect Fleet to Microsoft Entra, follow the instructions in
              the{" "}
              <CustomLink
                newTab
                text="guide"
                url="https://fleetdm.com/learn-more-about/connect-microsoft-entra"
              />
            </>
          }
        />
        <Card className={`${baseClass}__card`}>
          <p>
            You will need to copy and paste these values to create the
            application in Microsoft Entra.
          </p>
          <div className={`${baseClass}__url-inputs-wrapper`}>
            <InputField
              inputWrapperClass={`${baseClass}__url-input`}
              label="MDM terms of use URL"
              name="mdmTermsOfUseUrl"
              tooltip="The terms of use URL is used to display the terms of service to end users
              before turning on MDM for their host. The terms of use text informs users about
              policies that will be enforced on the host."
              value={generateMdmTermsOfUseUrl(
                config?.server_settings.server_url || ""
              )}
              enableCopy
            />
            <InputField
              inputWrapperClass={`${baseClass}__url-input`}
              label="MDM discovery URL"
              name="mdmDiscoveryUrl"
              tooltip="The enrollment URL is used to connect hosts with the MDM service."
              value={generateMdmDiscoveryUrl(
                config?.server_settings.server_url || ""
              )}
              enableCopy
            />
          </div>
        </Card>
      </>
    </MainContent>
  );
};

export default WindowsAutomaticEnrollmentPage;
