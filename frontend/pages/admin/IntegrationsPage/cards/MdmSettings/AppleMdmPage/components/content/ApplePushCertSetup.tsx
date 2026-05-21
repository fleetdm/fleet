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
          msg.toLowerCase().includes("download the certificate signing request")
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
      if (msg.includes("is not permitted for APNS certificate signing.")) {
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
        Follow the step-by-step guide to turn on Apple MDM.{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/turn-on-apple-mdm"
          text="Learn how"
          newTab
        />
      </p>
      <DownloadCSR baseClass={baseClass} onError={onDownloadError} />
      <FileUploader
        className={`${baseClass}__file-uploader ${
          isUploading ? `${baseClass}__file-uploader--loading` : ""
        }`}
        accept=".pem"
        buttonMessage={isUploading ? "Uploading..." : "Upload"}
        buttonType="brand-inverse-icon"
        disabled={isUploading}
        graphicName="file-pem"
        message="APNs certificate (.pem)"
        onFileUpload={onFileUpload}
      />
    </div>
  );
};

export default ApplePushCertSetup;
