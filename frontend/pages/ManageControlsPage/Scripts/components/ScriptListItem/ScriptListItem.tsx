import React, { useContext } from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { NotificationContext } from "context/notification";
import scriptAPI, { IScript } from "services/entities/scripts";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import ListItem from "components/ListItem";
import { ISupportedGraphicNames } from "components/ListItem/ListItem";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IScript;
  onDelete: (script: IScript) => void;
}

const getFileRenderDetails = (
  fileName: string
): { graphicName: ISupportedGraphicNames; platform: string | null } => {
  const fileExtension = fileName.split(".").pop();

  switch (fileExtension) {
    case "py":
      return { graphicName: "file-py", platform: null };
    case "sh":
      return { graphicName: "file-sh", platform: "macOS" };
    case "ps1":
      return { graphicName: "file-ps1", platform: "Windows" };
    default:
      return { graphicName: "file-script", platform: null };
  }
};

const ScriptListItem = ({ script, onDelete }: IScriptListItemProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onClickDownload = async () => {
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

  const { graphicName, platform } = getFileRenderDetails(script.name);

  const ListItemDetails = () => (
    <>
      {platform && <span>{platform}</span>}
      <span>{`Uploaded ${formatDistanceToNow(
        new Date(script.created_at)
      )} ago`}</span>
    </>
  );

  return (
    <ListItem
      className={baseClass}
      graphic={graphicName}
      title={script.name}
      details={<ListItemDetails />}
      actions={
        <>
          <Button
            className={`${baseClass}__action-button`}
            variant="text-icon"
            onClick={onClickDownload}
          >
            <Icon name="download" />
          </Button>
          <Button
            className={`${baseClass}__action-button`}
            variant="text-icon"
            onClick={() => onDelete(script)}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </>
      }
    />
  );
};

export default ScriptListItem;
