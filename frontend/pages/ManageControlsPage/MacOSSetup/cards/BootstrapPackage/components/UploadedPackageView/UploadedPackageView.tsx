import React, { useState } from "react";
import { useQuery } from "react-query";

import { IBootstrapPackage } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import UploadList from "pages/ManageControlsPage/components/UploadList";

import BootstrapPackageListItem from "../BootstrapPackageListItem";
import BootstrapPackageTable from "../BootstrapPackageTable/BootstrapPackageTable";
import DeletePackageModal from "../DeletePackageModal/DeletePackageModal";

const baseClass = "uploaded-package-view";

interface IUploadedPackageViewProps {
  currentTeamId: number;
  onDelete: () => void;
}

const UploadedPackageView = ({
  currentTeamId,
  onDelete,
}: IUploadedPackageViewProps) => {
  // TODO: hook up API call to get data
  const bootstrapPackage: IBootstrapPackage = {
    name: "test_package",
    team_id: 0,
    sha256: "123",
    token: "test-token",
    created_at: "2023-04-12T15:56:23Z", // TODO: add created at field.
  };

  return (
    <div className={baseClass}>
      <BootstrapPackageTable />
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
