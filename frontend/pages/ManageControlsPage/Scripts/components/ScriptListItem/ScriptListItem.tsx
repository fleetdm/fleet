import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";
import React, { useContext } from "react";

import { NotificationContext } from "context/notification";
import { IScript } from "interfaces/script";
import scriptAPI from "services/entities/scripts";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ListItem from "components/ListItem";
import { ISupportedGraphicNames } from "components/ListItem/ListItem";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IScript;
  onDelete: (script: IScript) => void;
  onClickScript: (script: IScript) => void;
  onEdit: (script: IScript) => void;
}

// TODO - useful to have a 'platform' field from API, for use elsewhere in app as well?
const getFileRenderDetails = (
  fileName: string
): { graphicName: ISupportedGraphicNames; platform: string | null } => {
  const fileExtension = fileName.split(".").pop();

  switch (fileExtension) {
    case "py":
      return { graphicName: "file-py", platform: null };
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

const onDownload = async (script: IScript, renderFlash: any) => {
  try {
    const content = await scriptAPI.downloadScript(script.id);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const filename = `${formatDate} ${script.name}`;
    const file = new File([content], filename);
    FileSaver.saveAs(file);
  } catch {
    renderFlash("error", "Couldn’t Download. Please try again.");
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
    <span>{`Uploaded ${formatDistanceToNow(new Date(createdAt))} ago`}</span>
  </div>
);

const ScriptListItem = ({
  script,
  onDelete,
  onClickScript,
  onEdit,
}: IScriptListItemProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const { graphicName, platform } = getFileRenderDetails(script.name);

  const onClickEdit = (evt: React.MouseEvent | React.KeyboardEvent) => {
    evt.stopPropagation();
    onEdit(script);
  };

  const onClickDownload = (evt: React.MouseEvent | React.KeyboardEvent) => {
    evt.stopPropagation();
    onDownload(script, renderFlash);
  };

  const onClickDelete = (evt: React.MouseEvent | React.KeyboardEvent) => {
    evt.stopPropagation();
    onDelete(script);
  };

  const actions = (
    <>
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onClickEdit}
            className={`${baseClass}__action-button`}
            variant="text-icon"
          >
            <Icon name="pencil" color="ui-fleet-black-75" />
          </Button>
        )}
      />
      <Button
        className={`${baseClass}__action-button`}
        variant="text-icon"
        onClick={onClickDownload}
      >
        <Icon name="download" />
      </Button>
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            onClick={onClickDelete}
            className={`${baseClass}__action-button`}
            variant="text-icon"
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        )}
      />
    </>
  );

  return (
    <ListItem
      className={baseClass}
      graphic={graphicName}
      title={<Button variant="text-link">{script.name}</Button>}
      details={
        <ScriptListItemDetails
          platform={platform}
          createdAt={script.created_at}
        />
      }
      actions={actions}
      onClick={() => onClickScript(script)}
    />
  );
};

export default ScriptListItem;
