import React, { useState } from "react";
import { useQuery } from "react-query";

import { IBootstrapPackageMetadata } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import UploadList from "pages/ManageControlsPage/components/UploadList";

import BootstrapPackageListItem from "../BootstrapPackageListItem";
import BootstrapPackageTable from "../BootstrapPackageTable/BootstrapPackageTable";
import DeletePackageModal from "../DeletePackageModal/DeletePackageModal";

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
        enroll to this team. Delete the package to upload a new one.
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
            onDelete={onDelete}
          />
        )}
      />
    </div>
  );
};

export default UploadedPackageView;
