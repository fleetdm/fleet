import React, { useContext } from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { NotificationContext } from "context/notification";
import scriptAPI, { IScript } from "services/entities/scripts";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import ListItem from "components/ListItem";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IScript;
  onDelete: (script: IScript) => void;
}

const getFileIconName = (fileName: string) => {
  const fileExtension = fileName.split(".").pop();

  switch (fileExtension) {
    case "py":
      return "file-py";
    case "sh":
      return "file-sh";
    default:
      return "file-script";
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

  return (
    <ListItem
      className={baseClass}
      graphic={getFileIconName(script.name)}
      title={script.name}
      details={
        <span>{`Uploaded ${formatDistanceToNow(
          new Date(script.created_at)
        )} ago`}</span>
      }
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
