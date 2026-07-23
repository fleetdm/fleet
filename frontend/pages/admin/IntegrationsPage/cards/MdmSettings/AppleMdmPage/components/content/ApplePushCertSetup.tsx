import React, { useCallback, useState } from "react";

import { getErrorReason } from "interfaces/errors";
import mdmAppleApi from "services/entities/mdm_apple";

import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import { notify } from "components/ToastNotification";
import DownloadCSR from "../../../../../../components/DownloadFileButtons/DownloadCSR";

interface IApplePushCertSetupProps {
  baseClass: string;
  onSetupSuccess: () => void;
}
const ApplePushCertSetup = ({
  baseClass,
  onSetupSuccess,
}: IApplePushCertSetupProps) => {
  const [isUploading, setIsUploading] = useState(false);

  const onFileUpload = useCallback(
    async (files: FileList | null) => {
      if (!files?.length) {
        notify.error("No file selected");
        return;
      }
      setIsUploading(true);
      try {
        await mdmAppleApi.uploadApplePushCertificate(files[0]);
        notify.success("MDM turned on successfully.");
        onSetupSuccess();
      } catch (e) {
        const msg = getErrorReason(e);
        if (msg.toLowerCase().includes("required private key")) {
          notify.error(
            <>
              Couldn&apos;t add APNs certificate. Please configure a private
              key.{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/fleet-server-private-key"
                text="Learn how"
                newTab
                variant="flash-message-link"
              />
            </>,
            { response: e }
          );
        } else {
          notify.error(msg || "Couldn’t connect. Please try again.", {
            response: e,
          });
        }
        setIsUploading(false);
      }
    },
    [onSetupSuccess]
  );

  const onDownloadError = useCallback((e: unknown) => {
    const msg = getErrorReason(e);
    if (msg.includes("is not permitted for APNS certificate signing.")) {
      notify.error(msg, { response: e });
    } else if (msg.toLowerCase().includes("required private key")) {
      notify.error(
        <>
          Couldn&apos;t download. Please configure a private key.{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/fleet-server-private-key"
            text="Learn how"
            newTab
            variant="flash-message-link"
          />
        </>,
        { response: e }
      );
    } else {
      notify.error("Something's gone wrong. Please try again.", {
        response: e,
      });
    }
  }, []);

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
        buttonType="secondary"
        disabled={isUploading}
        graphicName="file-pem"
        message="APNs certificate (.pem)"
        onFileUpload={onFileUpload}
      />
    </div>
  );
};

export default ApplePushCertSetup;
