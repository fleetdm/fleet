import React from "react";
import FileSaver from "file-saver";

import { uploadedFromNow } from "utilities/date_format";

import Button from "components/buttons/Button";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import { IGetSetupExperienceScriptResponse } from "services/entities/mdm";

const baseClass = "run-script-card";

interface IRunScriptCardProps {
  script: IGetSetupExperienceScriptResponse;
  onDelete: () => void;
}

const RunScriptCard = ({ script, onDelete }: IRunScriptCardProps) => {
  const onDownload = () => {
    const date = new Date();
    const filename = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}_${
      script.name
    }`;
    // TODO: download script
    // const file = new global.window.File(
    //   [JSON.stringify(script.enrollment_profile)],
    //   filename
    // );

    // FileSaver.saveAs(file);
  };

  return (
    <Card paddingSize="medium" className={baseClass}>
      <Graphic name="file-sh" />
      <div className={`${baseClass}__info`}>
        <span className={`${baseClass}__profile-name`}>{script.name}</span>
        <span className={`${baseClass}__uploaded-at`}>
          {uploadedFromNow(script.created_at)}
        </span>
      </div>
      <div className={`${baseClass}__actions`}>
        <Button
          className={`${baseClass}__download-button`}
          variant="text-icon"
          onClick={onDownload}
        >
          <Icon name="download" />
        </Button>
        <Button
          className={`${baseClass}__delete-button`}
          variant="text-icon"
          onClick={onDelete}
        >
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </div>
    </Card>
  );
};

export default RunScriptCard;
