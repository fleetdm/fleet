import React from "react";

import CustomLink from "components/CustomLink";
import FileUploader from "pages/ManageControlsPage/components/FileUploader";

const baseClass = "package-uploader";

interface IPackageUploaderProps {
  onUpload: () => void;
}

const PackageUploader = ({ onUpload }: IPackageUploaderProps) => {
  const onUploadFile = () => {
    // TODO hookup API
    onUpload();
  };

  return (
    <div className={baseClass}>
      <p>
        Upload a bootstrap package to install a configuration management tool
        (ex. Munki, Chef, or Puppet) on hosts that automatically enroll to
        Fleet.{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup"
          text="Learn more"
          newTab
        />
      </p>
      <FileUploader
        message="Package (.pkg)"
        icon="file-pkg"
        accept=".pkg"
        onFileUpload={onUploadFile}
      />
    </div>
  );
};

export default PackageUploader;
