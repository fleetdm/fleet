import React, { useContext } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import CustomLink from "components/CustomLink/CustomLink";
import InfoBanner from "components/InfoBanner";
import Icon from "components/Icon";

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
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-automatic-enrollment`}
        />
        <h1>Azure Active Directory</h1>
        <p className={`${baseClass}__description`}>
          The end user will see Microsoft&apos;s default initial setup. You can
          further simplify the initial device setup with Autopilot, which is
          similar to Apple&apos;s Automated Device Enrollment (DEP).{" "}
          <CustomLink
            newTab
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/setup-windows-mdm"
          />
        </p>
        {/* Ideally we'd use the native browser list styles and css to display
        the list numbers but this does not allow us to style the list items as we'd
        like so we write the numbers in the JSX instead. */}
        <ol className={`${baseClass}__setup-list`}>
          <li>
            <span>1.</span>
            <CustomLink
              newTab
              text="Sign in to Azure portal"
              url="https://fleetdm.com/sign-in-to/microsoft-automatic-enrollment-tool"
            />
          </li>
          <li>
            <span>2.</span>
            <p>
              At the top of the page, search “Domain names“ and select{" "}
              <b>Domain names</b>. Then select <b>+ Add custom domain</b>, type
              your Fleet URL (e.g. fleet.acme.com), and select <b>Add domain</b>
              .
            </p>
          </li>
          <li>
            <span>3.</span>
            <div>
              <p>
                Use the information presented in Azure AD to create a new TXT/MX
                record with your domain registrar, then select <b>Verify</b>.
              </p>
              <InfoBanner
                className={`${baseClass}__cloud-customer-banner`}
                color="purple"
                icon="warning"
              >
                <div className={`${baseClass}__banner-content`}>
                  <Icon name="error-outline" color="core-fleet-blue" />
                  <p>
                    If you&apos;re a managed-cloud customer, please reach out to
                    Fleet to create a TXT/MX record for you.
                  </p>
                </div>
              </InfoBanner>
            </div>
          </li>
          <li>
            <span>4.</span>
            <p>
              At the top of the page, search for “Mobility (MDM and MAM)“ and
              select <b>Mobility (MDM and MAM)</b>.
            </p>
          </li>
          <li>
            <span>5.</span>
            <p>
              Select <b>+ Add application</b>, then select{" "}
              <b>+ Create your own application</b>.
            </p>
          </li>
          <li>
            <span>6.</span>
            Enter “Fleet” as the name of your application and select{" "}
            <b>Create</b>.
          </li>
          <li>
            <span>7.</span>
            <div>
              <p>
                Set MDM user scope to <b>All</b>, then copy the URLs below,
                paste them in Azure AD, and select <b>Save</b>.
              </p>
              <div className={`${baseClass}__url-inputs-wrapper`}>
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="MDM terms of use URL"
                  name="mdmTermsOfUseUrl"
                  tooltip="The terms of use URL is used to display the terms of service to end users
                  before turning on MDM their host. The terms of use text informs users about
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
            </div>
          </li>
          <li>
            <span>8.</span>
            <p>
              Go back to <b>Mobility (MDM and MAM)</b>, refresh the page, then
              open newly created app and select{" "}
              <b>On-premises MDM application settings</b>.
            </p>
          </li>
          <li>
            <span>9.</span>
            <p>
              Select the link under <b>Application ID URI</b>, then select{" "}
              <b>Edit</b> button next to the Application ID URI input.
            </p>
          </li>
          <li>
            <span>10.</span>
            <p>
              Use your Fleet URL (e.g. fleet.acme.com) and select <b>Save</b>.
            </p>
          </li>
          <li>
            <span>11.</span>
            <p>
              Select <b>API permissions</b> from the sidebar, then select{" "}
              <b>+ Add a permission</b>.
            </p>
          </li>
          <li>
            <span>12.</span>
            <p>
              Select <b>Microsoft Graph</b>, then select{" "}
              <b>Delegated permissions</b>, and select{" "}
              <b>Group &gt; Group.Read.All</b> and{" "}
              <b>Group &gt; Group.ReadWrite.All</b>.
            </p>
          </li>
          <li>
            <span>13.</span>
            <div>
              Select <b>Application permissions</b>, then select following:
              <ul className={`${baseClass}__permissions-list`}>
                <li>Device &gt; Device.Read.All</li>
                <li>Device &gt; Device.ReadWrite.All</li>
                <li>Directory &gt; Directory.Read.All</li>
                <li>Group &gt; Group.Read.All</li>
                <li>User &gt; User.Read.All</li>
              </ul>
            </div>
          </li>
          <li>
            <span>14.</span>
            <p>
              Select <b>Add permissions</b>.
            </p>
          </li>
          <li>
            <span>15.</span>
            <p>
              Select <b>Grant admin consent for &lt;your tenant name&gt;</b>,
              and confirm.
            </p>
          </li>
          <li>
            <span>16.</span>
            <p>
              You&apos;re ready to automatically enroll Windows hosts to Fleet.
            </p>
          </li>
        </ol>
      </>
    </MainContent>
  );
};

export default WindowsAutomaticEnrollmentPage;
