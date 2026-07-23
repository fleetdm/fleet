import { format } from "date-fns";
import FileSaver from "file-saver";
import React from "react";

import { notify } from "components/ToastNotification";
import { IScript } from "interfaces/script";
import scriptAPI from "services/entities/scripts";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ListItem from "components/ListItem";
import { ISupportedGraphicNames } from "components/ListItem/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IScript;
  onDelete: (script: IScript) => void;
  onClickScript: (script: IScript) => void;
  onEdit: (script: IScript) => void;
  isTechnician?: boolean;
}

// TODO - useful to have a 'platform' field from API, for use elsewhere in app as well?
const getFileRenderDetails = (
  fileName: string
): { graphicName: ISupportedGraphicNames; platform: string | null } => {
  const fileExtension = fileName.split(".").pop();

  switch (fileExtension) {
    case "py":
      return { graphicName: "file-py", platform: "macOS & Linux" };
    case "sh":
      return { graphicName: "file-sh", platform: "macOS & Linux" };
    case "ps1":
      return { graphicName: "file-ps1", platform: "Windows" };
    default:
      return { graphicName: "file-script", platform: null };
  }
};

interface IScriptListItemDetailsProps {
  platform: string | null;
  createdAt: string;
}

const onDownload = async (script: IScript) => {
  try {
    const content = await scriptAPI.downloadScript(script.id);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const filename = `${formatDate} ${script.name}`;
    const file = new File([content], filename);
    FileSaver.saveAs(file);
  } catch (e) {
    notify.error("Couldn’t Download. Please try again.", { response: e });
  }
};

const ScriptListItemDetails = ({
  platform,
  createdAt,
}: IScriptListItemDetailsProps) => (
  <div className={`${baseClass}__details`}>
    {platform && (
      <>
        <span>{platform}</span>
        <span>&bull;</span>
      </>
    )}
    <span>
      Uploaded <HumanTimeDiffWithDateTip timeString={createdAt} />
    </span>
  </div>
);

const ScriptListItem = ({
  script,
  onDelete,
  onClickScript,
  onEdit,
  isTechnician,
}: IScriptListItemProps) => {
  const { graphicName, platform } = getFileRenderDetails(script.name);

  const onClickEdit = () => {
    onEdit(script);
  };

  const onClickDownload = () => {
    onDownload(script);
  };

  const onClickDelete = () => {
    onDelete(script);
  };

  const actions = (
    <div
      className={`${baseClass}__actions`}
      onClick={(evt) => evt.stopPropagation()}
    >
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onClickEdit}
            className={`${baseClass}__action-button`}
            variant="secondary"
            ariaLabel={`Edit ${script.name}`}
          >
            <Icon name="pencil" />
          </Button>
        )}
      />
      <Button
        className={`${baseClass}__action-button`}
        variant="secondary"
        onClick={onClickDownload}
        ariaLabel={`Download ${script.name}`}
      >
        <Icon name="download" />
      </Button>
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onClickDelete}
            className={`${baseClass}__action-button`}
            variant="secondary"
            ariaLabel={`Delete ${script.name}`}
          >
            <Icon name="trash" />
          </Button>
        )}
      />
    </div>
  );

  return (
    <ListItem
      className={baseClass}
      graphic={graphicName}
      title={
        <TooltipWrapper
          tipContent={`ID: ${script.id}`}
          underline={false}
          position="top"
          showArrow
        >
          <Button variant="link" className={`${baseClass}__title-button`}>
            <TooltipTruncatedText value={script.name} fixedPositionStrategy />
          </Button>
        </TooltipWrapper>
      }
      details={
        <ScriptListItemDetails
          platform={platform}
          createdAt={script.created_at}
        />
      }
      actions={isTechnician ? undefined : actions}
      onClick={() => onClickScript(script)}
    />
  );
};

export default ScriptListItem;
