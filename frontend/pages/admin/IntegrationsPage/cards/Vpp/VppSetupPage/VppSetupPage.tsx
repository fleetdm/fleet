import React, { useContext, useState } from "react";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import { InjectedRouter } from "react-router";
import { getErrorReason } from "interfaces/errors";

const baseClass = "vpp-setup-page";

interface IVppSetupContentProps {
  router: InjectedRouter;
}

const VPPSetupContent = ({ router }: IVppSetupContentProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  const uploadToken = async (data: FileList | null) => {
    setIsUploading(true);
    const token = data?.[0];
    if (!token) {
      setIsUploading(false);
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      // TODO: API integration
      // await mdmAppleBmAPI.uploadToken(token);
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT);
    } catch (e) {
      // TODO: error messages
      const msg = getErrorReason(e);
      if (msg.toLowerCase().includes("valid token")) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn't Upload. Please try again.");
      }
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className={`${baseClass}__setup-content`}>
      <p>
        Connect Fleet to your Apple Business Manager account to enable access to
        purchased apps.
      </p>
      <ol className={`${baseClass}__setup-list`}>
        <li>
          <span>1.</span>
          <p>
            Sign in to{" "}
            <CustomLink
              newTab
              url="https://business.apple.com"
              text="Apple Business Manager"
            />
            <br />
            If your organization doesn&apos;t have an account, select{" "}
            <b>Sign up now</b>.
          </p>
        </li>
        <li>
          <span>2.</span>
          <p>
            Select your <b>account name</b> at the bottom left of the screen,
            then select <b>Preferences</b>.
          </p>
        </li>
        <li>
          <span>3.</span>
          <p>Select Payments and Billings in the menu.</p>
        </li>
        <li>
          <span>4.</span>
          <p>
            Under the <b>Content Tokens</b>, download the token for the location
            you want to use. Each token is based on a location in Apple Business
            Manager.
          </p>
        </li>
        <li>
          <span>5.</span>
          <p>Upload content token (.vpptoken file) below.</p>
        </li>
      </ol>
      <FileUploader
        className={`${baseClass}__file-uploader ${
          isUploading ? `${baseClass}__file-uploader--loading` : ""
        }`}
        accept=".vpptoken"
        message="Content token (.vpptoken)"
        graphicName="file-vpp"
        buttonType="link"
        buttonMessage={isUploading ? "Uploading..." : "Upload"}
        onFileUpload={uploadToken}
      />
    </div>
  );
};

const VPPDisableOrRenewContent = () => {
  return <>disable</>;
};

interface IVppSetupPageProps {
  router: InjectedRouter;
}

const VppSetupPage = ({ router }: IVppSetupPageProps) => {
  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to automatic enrollment"
          path={PATHS.ADMIN_INTEGRATIONS_VPP}
          className={`${baseClass}__back-to-vpp`}
        />
        <h1>Volume Purchasing Program (VPP)</h1>
        {true ? (
          <VPPSetupContent router={router} />
        ) : (
          <VPPDisableOrRenewContent />
        )}
      </>
    </MainContent>
  );
};

export default VppSetupPage;
