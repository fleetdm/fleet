import React from "react";

import { IEulaMetadataResponse } from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import UploadList from "pages/ManageControlsPage/components/UploadList";
import EulaListItem from "../EulaListItem/EulaListItem";

const baseClass = "uploaded-eula-view";

interface IUploadedEulaViewProps {
  eulaMetadata: IEulaMetadataResponse;
  onDelete: () => void;
}

const UploadedEulaView = ({
  eulaMetadata,
  onDelete,
}: IUploadedEulaViewProps) => {
  return (
    <div className={baseClass}>
      <p>
        Require end users to agree to a EULA when they first setup their new
        macOS hosts.{" "}
        <CustomLink
          url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula"
          text="Learn more"
          newTab
        />
      </p>
      <UploadList
        keyAttribute="name"
        listItems={[eulaMetadata]}
        ListItemComponent={({ listItem }) => (
          <EulaListItem eulaData={listItem} onDelete={onDelete} />
        )}
      />
    </div>
  );
};

export default UploadedEulaView;
