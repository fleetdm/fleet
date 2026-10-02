import React from "react";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Graphic from "components/Graphic";
import TooltipWrapper from "components/TooltipWrapper";
import { IEulaMetadataResponse } from "services/entities/mdm";
import { timeAgo } from "utilities/date_format";
import endpoints from "utilities/endpoints";

import { EULA_PLATFORM_CONFIG, EulaPlatform } from "../../helpers";

const baseClass = "eula-list-item";

interface IEulaListItemProps {
  platform: EulaPlatform;
  eulaData: IEulaMetadataResponse;
  onDelete: () => void;
  onShowExample: () => void;
}

const EulaListItem = ({
  platform,
  eulaData,
  onDelete,
  onShowExample,
}: IEulaListItemProps) => {
  const config = EULA_PLATFORM_CONFIG[platform];

  const onOpenEula = () => {
    window.open(`/api${endpoints.MDM_EULA(eulaData.token)}`, "_blank");
  };

  const renderPrimaryAction = () => {
    if (config.hasExample) {
      return (
        <Button variant="secondary" onClick={onShowExample}>
          Example EULA
        </Button>
      );
    }
    return (
      <TooltipWrapper
        tipContent="Preview"
        underline={false}
        showArrow
        position="top"
      >
        <Button
          variant="secondary"
          icon="external-link"
          onClick={onOpenEula}
          ariaLabel="Preview EULA"
        />
      </TooltipWrapper>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__list-item-data`}>
        <Graphic name={config.graphicName} />
        <div className={`${baseClass}__list-item-info`}>
          <span className={`${baseClass}__list-item-name`}>
            {eulaData.name}
          </span>
          <span className={`${baseClass}__list-item-uploaded`}>
            {`Uploaded ${timeAgo(new Date(eulaData.created_at), {
              addSuffix: true,
            })}`}
          </span>
        </div>
      </div>

      <div
        className={`${baseClass}__value-group ${baseClass}__list-item-actions`}
      >
        {renderPrimaryAction()}
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              tipContent="Delete"
              disableTooltip={disableChildren}
              underline={false}
              showArrow
              position="top"
            >
              <Button
                variant="secondary"
                icon="trash"
                onClick={onDelete}
                disabled={disableChildren}
                ariaLabel="Delete EULA"
              />
            </TooltipWrapper>
          )}
        />
      </div>
    </div>
  );
};

export default EulaListItem;
