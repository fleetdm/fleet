import React, { useContext, useEffect, useState } from "react";

import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import Icon from "components/Icon";
import DownloadCSR from "../actions/DownloadCSR";

interface IApplePushCertSetupProps {
  baseClass: string;
  // onClickRequest: () => void;
}
const ApplePushCertSetup = ({ baseClass }: IApplePushCertSetupProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  const onFileUpload = (files: FileList | null) => {
    console.log(files?.length);
    setIsUploading(true);
    // TODO: upload file
  };

  // TODO: remove this (it's just for dev testing)
  useEffect(() => {
    if (isUploading) {
      setTimeout(() => {
        setIsUploading(false);
      }, 5000);
    }
  }, [isUploading]);

  return (
    <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
      <p className={`${baseClass}__setup-description`}>
        Connect Fleet to Apple Push Certificates Portal to turn on MDM.
      </p>
      <ol className={`${baseClass}__setup-instructions-list`}>
        <li>
          <p>
            1.Download a certificate signing request (CSR) for Apple Push
            Notification service (APNs).
          </p>
          <DownloadCSR baseClass={baseClass} />
        </li>
        <li>
          <p>
            2.{" "}
            <CustomLink
              url="https://identity.apple.com/pushcert/"
              text="Sign in to Apple Push Certificates Portal"
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
              // accept=".pem" // TODO: uncomment this
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
