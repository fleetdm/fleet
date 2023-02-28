import React from "react";
import { formatDistanceToNow } from "date-fns";

import { IMdmScript } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "script-list-item";

interface IScriptListItemProps {
  script: IMdmScript;
  onRerun: (script: IMdmScript) => void;
  onDelete: (script: IMdmScript) => void;
}

const getStatusClassName = (value: number) => {
  return value !== 0 ? `${baseClass}__has-value` : "";
};

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

const ScriptListItem = ({
  script,
  onRerun,
  onDelete,
}: IScriptListItemProps) => {
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
      <div
        className={`${baseClass}__value-group ${baseClass}__script-statuses`}
      >
        <span className={getStatusClassName(script.ran)}>{script.ran}</span>
        <span className={getStatusClassName(script.pending)}>
          {script.pending}
        </span>
        <span className={getStatusClassName(script.errors)}>
          {script.errors}
        </span>
      </div>

      <div className={`${baseClass}__value-group ${baseClass}__script-actions`}>
        <Button
          className={`${baseClass}__refresh-button`}
          variant="text-icon"
          onClick={() => onRerun(script)}
        >
          <Icon name="refresh" />
        </Button>
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
