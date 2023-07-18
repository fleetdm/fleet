import React from "react";

import PATHS from "router/paths";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import CustomLink from "components/CustomLink/CustomLink";

const baseClass = "windows-automatic-enrollment-page";

interface IWindowsAutomaticEnrollmentPageProps {}

const WindowsAutomaticEnrollmentPage = ({}: IWindowsAutomaticEnrollmentPageProps) => {
  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to automatic enrollment"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-automatic-enrollment`}
        />
        <h1>Azure Active Directory</h1>
        <p>
          Connect to Azure AD to automatically enroll Windows hosts.{" "}
          <CustomLink
            newTab
            text="Learn more"
            url="https://fleetdm.com/docs/using-fleet/windows-mdm-setup"
          />
        </p>
        <ol className={`${baseClass}__setup-list`}>
          <li>
            <CustomLink
              newTab
              text="Sign in to Azure portal"
              url="https://azure.microsoft.com/en-gb/get-started/azure-portal"
            />
          </li>
          <li>
            Select <b>Azure Active Directory &gt; Custom domain names</b>, then
            select <b>+ Add custom domain</b>, type your organization&apos;s
            domain name (e.g. acme.com), and select <b>Add domain</b>.
          </li>
          <li>
            Use the information presented in Azure AD to create a new TXT/MX
            record with your domain registrar, then select <b>Verify</b>.
          </li>
          <li>
            Select <b>Azure Active Directory &gt; Mobility (MDM and MAM)</b> (or
            search for “Mobility” at the top of the page).
          </li>
          <li>
            Select <b>+ Add application</b>, then select{" "}
            <b>+ Create your own application</b>.
          </li>
          <li>
            Enter “Fleet” as the name of your application and select{" "}
            <b>Create</b>.
          </li>
          <li>
            Set MDM user scope to <b>All</b>, then copy the URLs below, paste
            them in Azure AD, and select <b>Save</b>.
          </li>
          <li>
            Go back to <b>Mobility (MDM and MAM)</b>, refresh the page, then
            open newly created app On-premises MDM application settings
            <b>On-premises MDM application settings</b>.
          </li>
          <li>
            Select the link under <b>Application ID URI</b>, then select{" "}
            <b>Edit</b> button next to the Application ID URI input.
          </li>
          <li>
            Use your organization&apos;s domain you added in the first step, and
            select <b>Save</b>.
          </li>
          <li>
            Select <b>API permissions</b> from the sidebar, then select{" "}
            <b>+ Add permissions</b>.
          </li>
          <li>
            Select <b>Microsoft Graph</b>, then select{" "}
            <b>Delegated permissions</b>, and select{" "}
            <b>Group &gt; Group.Read.All</b> and{" "}
            <b>Group &gt; Group.ReadWrite.All</b>.
          </li>
          <li>
            Select <b>Application permissions</b>, then select following:
            <ul>
              <li>Device.Read.All</li>
              <li>Device &gt; Device.ReadWrite.All</li>
              <li>Directory &gt; Directory.Read.All</li>
              <li>Group &gt; Group.Read.All</li>
              <li>User &gt; User.Read.All</li>
            </ul>
          </li>
          <li>
            Select <b>Add permissions</b>.
          </li>
          <li>
            Select <b>Grant admin consent for &lt;your tenant name&gt;</b>, and
            confirm.
          </li>
          <li>
            You&apos;re ready to automatically enroll Windows hosts to Fleet.
          </li>
        </ol>
      </>
    </MainContent>
  );
};

export default WindowsAutomaticEnrollmentPage;
