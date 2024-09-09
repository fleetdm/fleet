import React from "react";
import { formatDistanceToNow } from "date-fns";

import endpoints from "utilities/endpoints";
import { IEulaMetadataResponse } from "services/entities/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import Graphic from "components/Graphic";

const baseClass = "eula-list-item";

interface IEulaListItemProps {
  eulaData: IEulaMetadataResponse;
  onDelete: () => void;
}

const EulaListItem = ({ eulaData, onDelete }: IEulaListItemProps) => {
  const onOpenEula = () => {
    window.open(`/api/${endpoints.MDM_EULA(eulaData.token)}`, "_blank");
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__list-item-data`}>
        <Graphic name="file-pdf" />
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
          onClick={onOpenEula}
        >
          <Icon
            name="external-link"
            size="medium"
            className={`${baseClass}__external-icon`}
            color="ui-fleet-black-75"
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
