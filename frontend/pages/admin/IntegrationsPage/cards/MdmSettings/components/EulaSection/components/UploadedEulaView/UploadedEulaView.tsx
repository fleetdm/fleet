import React from "react";

import UploadList from "components/UploadList";
import { IEulaMetadataResponse } from "services/entities/mdm";

import { EulaPlatform } from "../../helpers";
import EulaListItem from "../EulaListItem/EulaListItem";

const baseClass = "uploaded-eula-view";

interface IUploadedEulaViewProps {
  platform: EulaPlatform;
  eulaMetadata: IEulaMetadataResponse;
  onDelete: () => void;
  onShowExample: () => void;
}

const UploadedEulaView = ({
  platform,
  eulaMetadata,
  onDelete,
  onShowExample,
}: IUploadedEulaViewProps) => {
  return (
    <div className={baseClass}>
      <UploadList
        keyAttribute="name"
        listItems={[eulaMetadata]}
        ListItemComponent={({ listItem }) => (
          <EulaListItem
            platform={platform}
            eulaData={listItem}
            onDelete={onDelete}
            onShowExample={onShowExample}
          />
        )}
      />
    </div>
  );
};

export default UploadedEulaView;
