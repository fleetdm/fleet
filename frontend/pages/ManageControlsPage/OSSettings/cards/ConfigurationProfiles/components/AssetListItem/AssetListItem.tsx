import React from "react";
import { format } from "date-fns";
import FileSaver from "file-saver";

import { timeAgo } from "utilities/date_format";
import { IMdmAsset } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";
import { notify } from "components/ToastNotification";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import Icon from "components/Icon";
import ListItem from "components/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "asset-list-item";

interface IAssetListItemProps {
  asset: IMdmAsset;
  onClickDelete: (asset: IMdmAsset) => void;
  isTechnician?: boolean;
}

const AssetDetails = ({ asset }: { asset: IMdmAsset }) => {
  const uploadedAt = asset.uploaded_at ? new Date(asset.uploaded_at) : null;
  const uploadedText =
    !uploadedAt || Number.isNaN(uploadedAt.getTime())
      ? "Uploaded"
      : `Uploaded ${timeAgo(uploadedAt, { addSuffix: true })}`;

  return (
    <div className={`${baseClass}__details`}>
      <span>{uploadedText}</span>
      <span>&bull;</span>
      <span className={`${baseClass}__identifier`}>{asset.identifier}</span>
      <CopyButton
        copyText={asset.identifier}
        variant="compact"
        ariaLabel={`Copy ${asset.identifier}`}
      />
    </div>
  );
};

const AssetListItem = ({
  asset,
  onClickDelete,
  isTechnician,
}: IAssetListItemProps) => {
  const onClickDownload = async () => {
    try {
      const content = await mdmAPI.downloadAsset(asset.asset_uuid);
      const formatDate = format(new Date(), "yyyy-MM-dd");
      const fileContent = JSON.stringify(content, null, 2);
      const file = new File([fileContent], `${formatDate}_${asset.name}.json`);
      FileSaver.saveAs(file);
    } catch (e) {
      notify.error("Couldn't download. Please try again.", { response: e });
    }
  };

  const actions = (
    <>
      <Button
        className={`${baseClass}__action-button`}
        variant="secondary"
        onClick={onClickDownload}
        ariaLabel={`Download ${asset.name}`}
      >
        <Icon name="download" />
      </Button>
      {!isTechnician && (
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              className={`${baseClass}__action-button`}
              variant="secondary"
              onClick={() => onClickDelete(asset)}
              ariaLabel={`Delete ${asset.name}`}
            >
              <Icon name="trash" />
            </Button>
          )}
        />
      )}
    </>
  );

  return (
    <ListItem
      className={baseClass}
      graphic="file-json"
      title={
        <TooltipWrapper
          tipContent={`UUID: ${asset.asset_uuid}`}
          underline={false}
          position="top"
          showArrow
        >
          <TooltipTruncatedText value={asset.name} />
        </TooltipWrapper>
      }
      details={<AssetDetails asset={asset} />}
      actions={actions}
    />
  );
};

export default AssetListItem;
