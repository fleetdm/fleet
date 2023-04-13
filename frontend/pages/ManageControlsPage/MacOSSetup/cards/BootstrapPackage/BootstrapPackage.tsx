import React from "react";

import { IBootstrapPackage } from "interfaces/mdm";

import CustomLink from "components/CustomLink/CustomLink";
import UploadList from "pages/ManageControlsPage/components/UploadList";
import FileUploader from "pages/ManageControlsPage/components/FileUploader/FileUploader";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview/BootstrapPackagePreview";
import BootstrapPackageListItem from "./components/BootstrapPackageListItem/BootstrapPackageListItem";

const baseClass = "bootstrap-package";

const BootstrapPackageUpload = () => {
  return (
    <FileUploader
      message="Package (.pkg)"
      icon="file-pkg"
      accept=".pkg"
      onFileUpload={() => {}}
    />
  );
};

interface IBootstrapPackageProps {
  currentTeamId?: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  const bootstrapPackage: IBootstrapPackage = {
    name: "test_package",
    team_id: 0,
    sha256: "123",
    token: "test-token",
    created_at: "2023-04-12T15:56:23Z", // TODO: add created at field.
  };

  const onClickDelete = () => {};

  return (
    <div className={baseClass}>
      <h2>Bootstrap package</h2>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__uploader-table-container`}>
          <p>
            Upload a bootstrap package to install a configuration management
            tool (ex. Munki, Chef, or Puppet) on hosts that automatically enroll
            to Fleet.{" "}
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup"
              text="Learn more"
              newTab
            />
          </p>
          <UploadList
            listItems={[bootstrapPackage]}
            ListItemComponent={({ listItem }) => (
              <BootstrapPackageListItem
                bootstrapPackage={listItem}
                onDelete={onClickDelete}
              />
            )}
          />
          <BootstrapPackageUpload />
        </div>
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
      </div>
    </div>
  );
};

export default BootstrapPackage;
