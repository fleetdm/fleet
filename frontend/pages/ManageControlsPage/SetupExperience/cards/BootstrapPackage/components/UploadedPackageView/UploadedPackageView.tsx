import React from "react";

import { IBootstrapPackageMetadata } from "interfaces/mdm";

import CustomLink from "components/CustomLink";
import UploadList from "pages/ManageControlsPage/components/UploadList";

import BootstrapPackageListItem from "../BootstrapPackageListItem";
import BootstrapPackageTable from "../BootstrapPackageTable/BootstrapPackageTable";

const baseClass = "uploaded-package-view";

interface IUploadedPackageViewProps {
  bootstrapPackage: IBootstrapPackageMetadata;
  currentTeamId: number;
  onDelete: () => void;
}

const UploadedPackageView = ({
  bootstrapPackage,
  currentTeamId,
  onDelete,
}: IUploadedPackageViewProps) => {
  return (
    <div className={baseClass}>
      <BootstrapPackageTable currentTeamId={currentTeamId} />
      <p>
        This package is automatically installed on hosts that automatically
        enroll to this team. Delete the package to upload a new one.{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience"
          text="Learn more"
          newTab
        />
      </p>
      <UploadList
        keyAttribute="name"
        listItems={[bootstrapPackage]}
        ListItemComponent={({ listItem }) => (
          <BootstrapPackageListItem
            bootstrapPackage={listItem}
            onDelete={onDelete}
          />
        )}
      />
    </div>
  );
};

export default UploadedPackageView;
