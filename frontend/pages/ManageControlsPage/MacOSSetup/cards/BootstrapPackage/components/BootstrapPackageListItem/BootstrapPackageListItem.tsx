import React from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { IBootstrapPackageMetadata } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "bootstrap-package-list-item";

interface IBootstrapPackageListItemProps {
  bootstrapPackage: IBootstrapPackageMetadata;
  onDelete: (bootstrapPackage: IBootstrapPackageMetadata) => void;
}

const BootstrapPackageListItem = ({
  bootstrapPackage,
  onDelete,
}: IBootstrapPackageListItemProps) => {
  const onClickDownload = async () => {
    const fileContent = await mdmAPI.downloadBootstrapPackage(
      bootstrapPackage.token
    );
    const file = new File([fileContent], bootstrapPackage.name);
    FileSaver.saveAs(file);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__list-item-data`}>
        <Icon name="file-pkg" />
        <div className={`${baseClass}__list-item-info`}>
          <span className={`${baseClass}__list-item-name`}>
            {bootstrapPackage.name}
          </span>
          <span className={`${baseClass}__list-item-uploaded`}>
            {`Uploaded ${formatDistanceToNow(
              new Date(bootstrapPackage.created_at)
            )} ago`}
          </span>
        </div>
      </div>

      <div
        className={`${baseClass}__value-group ${baseClass}__list-item-actions`}
      >
        <Button
          className={`${baseClass}__list-item-button`}
          variant="text-icon"
          onClick={onClickDownload}
        >
          <Icon name="download" />
        </Button>
        <Button
          className={`${baseClass}__list-item-button`}
          variant="text-icon"
          onClick={() => onDelete(bootstrapPackage)}
        >
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </div>
    </div>
  );
};

export default BootstrapPackageListItem;
