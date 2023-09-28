import React from "react";
import { formatDistanceToNow } from "date-fns";

import { IScript } from "services/entities/scripts";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IScript;
  onDelete: (script: IScript) => void;
}

const getFileIconName = (fileName: string) => {
  const fileExtension = fileName.split(".").pop();

  switch (fileExtension) {
    case "py":
      return "file-python";
    case "zsh":
      return "file-zsh";
    case "sh":
      return "file-bash";
    default:
      return "file-generic";
  }
};

const ScriptListItem = ({ script, onDelete }: IScriptListItemProps) => {
  const onClickDownload = () => {
    console.log("download");
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__script-data`}>
        <Icon name={getFileIconName(script.name)} />
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
