import React from "react";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "upload-list-heading";

interface IUploadListHeadingProps {
  entityName: string;
  createEntityText: string;
  onClickAdd?: () => void;
}

const UploadListHeading = ({
  entityName,
  createEntityText,
  onClickAdd,
}: IUploadListHeadingProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__upload-name-heading`}>{entityName}</span>
      <span className={`${baseClass}__actions-heading`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="brand-inverse-icon"
              className={`${baseClass}__add-button`}
              onClick={onClickAdd}
              iconStroke
            >
              <>
                <Icon name="plus" color="core-fleet-green" />
                {createEntityText}
              </>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default UploadListHeading;
