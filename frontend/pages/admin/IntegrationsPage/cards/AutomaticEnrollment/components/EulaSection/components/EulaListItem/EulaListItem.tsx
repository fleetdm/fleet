import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { formatDistanceToNow } from "date-fns";
import React from "react";

import { IEulaMetadataResponse } from "services/entities/mdm";

const baseClass = "eula-list-item";

interface IEulaListItemProps {
  eulaData: IEulaMetadataResponse;
  onDelete: () => void;
}

const EulaListItem = ({ eulaData, onDelete }: IEulaListItemProps) => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__list-item-data`}>
        <Icon name="file-pdf" />
        <div className={`${baseClass}__list-item-info`}>
          <span className={`${baseClass}__list-item-name`}>
            {eulaData.name}
          </span>
          <span className={`${baseClass}__list-item-uploaded`}>
            {`Uploaded ${formatDistanceToNow(
              new Date(eulaData.created_at)
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
          // onClick={() => window.open(eulaData, "_blank")}
          onClick={() => console.log("opening")}
        >
          <Icon
            name="external-link"
            size="medium"
            className={`${baseClass}__external-icon`}
            color={"ui-fleet-black-75"}
          />
        </Button>
        <Button
          className={`${baseClass}__list-item-button`}
          variant="text-icon"
          onClick={() => onDelete()}
        >
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </div>
    </div>
  );
};

export default EulaListItem;
