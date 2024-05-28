import React, { useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import mdmAppleApi from "services/entities/mdm_apple";

import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import DownloadCSR from "../actions/DownloadCSR";

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
        renderFlash("success", "macOS MDM turned on successfully.");
        onSetupSuccess();
      } catch (e) {
        const msg = getErrorReason(e);
        if (msg.toLowerCase().includes("invalid certificate")) {
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
      } else {
        renderFlash("error", "Something’s gone wrong. Please try again.");
      }
    },
    [renderFlash]
  );

  // TODO: Cleanup styles to indent list items properly (first line hangs over indented content below)

  return (
    <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
      <p className={`${baseClass}__setup-description`}>
        Connect Fleet to Apple Push Certificates Portal to turn on MDM.
      </p>
      <ol className={`${baseClass}__setup-instructions-list`}>
        <li>
          <p>
            1. Download a certificate signing request (CSR) for Apple Push
            Notification service (APNs).
          </p>
          <DownloadCSR baseClass={baseClass} onError={onDownloadError} />
        </li>
        <li>
          <p>
            2. Sign in to{" "}
            <CustomLink
              url="https://identity.apple.com/pushcert/"
              text="Apple Push Certificates Portal"
              newTab
            />
            <br />
            If you don&apos;t have an Apple ID, select <b>Create yours now</b>.
          </p>
        </li>
        <li>
          <p>
            3. In Apple Push Certificates Portal, select{" "}
            <b>Create a Certificate</b>, upload your CSR, and download your APNs
            certificate.
          </p>
        </li>
        <li>
          <p>
            4. Upload APNs certificate (.pem file) below.
            <FileUploader
              className={`${baseClass}__file-uploader ${
                isUploading ? `${baseClass}__file-uploader--loading` : ""
              }`}
              accept=".pem"
              buttonMessage={isUploading ? "Uploading..." : "Upload"}
              buttonType="link"
              diabled={isUploading}
              graphicName="file-pem"
              message="APNs certificate (.pem)"
              onFileUpload={onFileUpload}
            />
          </p>
        </li>
      </ol>
    </div>
  );
};

export default ApplePushCertSetup;
