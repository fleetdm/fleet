import React, { useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import mdmAppleApi from "services/entities/mdm_apple";

import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import ClickableUrls from "components/ClickableUrls";
import DownloadCSR from "../../../../../../components/DownloadFileButtons/DownloadCSR";

interface IApplePushCertSetupProps {
  baseClass: string;
  onSetupSuccess: () => void;
}
const ApplePushCertSetup = ({
  baseClass,
  onSetupSuccess,
}: IApplePushCertSetupProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  const onFileUpload = useCallback(
    async (files: FileList | null) => {
      if (!files?.length) {
        renderFlash("error", "No file selected");
        return;
      }
      setIsUploading(true);
      try {
        await mdmAppleApi.uploadApplePushCertificate(files[0]);
        renderFlash("success", "MDM turned on successfully.");
        onSetupSuccess();
      } catch (e) {
        const msg = getErrorReason(e);
        if (
          msg.toLowerCase().includes("invalid certificate") ||
          msg.toLowerCase().includes("required private key")
        ) {
          renderFlash("error", msg);
        } else {
          renderFlash("error", "Couldn’t connect. Please try again.");
        }
        setIsUploading(false);
      }
    },
    [renderFlash, onSetupSuccess]
  );

  const onDownloadError = useCallback(
    (e: unknown) => {
      const msg = getErrorReason(e);
      if (msg.toLowerCase().includes("email address")) {
        renderFlash("error", msg);
      } else if (msg.toLowerCase().includes("required private key")) {
        // replace link with actually clickable link
        renderFlash("error", ClickableUrls({ text: msg }));
      } else {
        renderFlash("error", "Something's gone wrong. Please try again.");
      }
    },
    [renderFlash]
  );

  return (
    <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
      <p className={`${baseClass}__setup-description`}>
        Connect Fleet to Apple Push Certificates Portal to turn on MDM.
      </p>
      <div>
        <ol className={`${baseClass}__setup-instructions-list`}>
          <li>
            <span>1. </span>
            <span>
              <span>
                Download a certificate signing request (CSR) for Apple Push
                Notification service (APNs).
              </span>
              <DownloadCSR baseClass={baseClass} onError={onDownloadError} />
            </span>
          </li>
          <li>
            <span>2. </span>
            <span>
              Sign in to{" "}
              <CustomLink
                url="https://identity.apple.com/pushcert/"
                text="Apple Push Certificates Portal"
                newTab
              />
              <br />
              If you don&apos;t have an Apple ID, select <b>Create yours now</b>
              .
            </span>
          </li>
          <li>
            <span>3. </span>
            <span>
              In Apple Push Certificates Portal, select{" "}
              <b>Create a Certificate</b>, upload your CSR, and download your
              APNs certificate.
            </span>
          </li>
          <li>
            <span>4. </span>
            <span>Upload APNs certificate (.pem file) below.</span>
          </li>
        </ol>
        <FileUploader
          className={`${baseClass}__file-uploader ${
            isUploading ? `${baseClass}__file-uploader--loading` : ""
          }`}
          accept=".pem"
          buttonMessage={isUploading ? "Uploading..." : "Upload"}
          buttonType="link"
          disabled={isUploading}
          graphicName="file-pem"
          message="APNs certificate (.pem)"
          onFileUpload={onFileUpload}
        />
      </div>
    </div>
  );
};

export default ApplePushCertSetup;
