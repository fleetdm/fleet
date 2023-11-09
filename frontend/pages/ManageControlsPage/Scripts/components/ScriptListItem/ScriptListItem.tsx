import React, { useContext } from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { NotificationContext } from "context/notification";
import scriptAPI, { IScript } from "services/entities/scripts";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import Graphic from "components/Graphic";

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
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__script-data`}>
        <Graphic name={getFileIconName(script.name)} />
        <div className={`${baseClass}__script-info`}>
          <span className={`${baseClass}__script-name`}>{script.name}</span>
          <span className={`${baseClass}__script-uploaded`}>
            {`Uploaded ${formatDistanceToNow(new Date(script.created_at))} ago`}
          </span>
        </div>
      </div>
      <div className={`${baseClass}__value-group ${baseClass}__script-actions`}>
        <Button
          className={`${baseClass}__download-button`}
          variant="text-icon"
          onClick={onClickDownload}
        >
          <Icon name="download" />
        </Button>
        <Button
          className={`${baseClass}__delete-button`}
          variant="text-icon"
          onClick={() => onDelete(script)}
        >
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </div>
    </div>
  );
};

export default ScriptListItem;
