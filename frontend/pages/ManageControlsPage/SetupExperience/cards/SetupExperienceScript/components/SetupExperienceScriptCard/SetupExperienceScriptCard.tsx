import React, { useContext } from "react";
import FileSaver from "file-saver";

import mdmAPI, {
  IGetSetupExperienceScriptResponse,
} from "services/entities/mdm";

import { uploadedFromNow } from "utilities/date_format";

import Button from "components/buttons/Button";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import { API_NO_TEAM_ID } from "interfaces/team";

const baseClass = "setup-experience-script-card";

interface ISetupExperienceScriptCardProps {
  script: IGetSetupExperienceScriptResponse;
  onDelete: () => void;
}

const SetupExperienceScriptCard = ({
  script,
  onDelete,
}: ISetupExperienceScriptCardProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onDownload = async () => {
    try {
      const teamId = script.team_id ?? API_NO_TEAM_ID;
      await mdmAPI.downloadSetupExperienceScript(teamId);
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    // TODO: download script integration

    // const date = new Date();
    // const filename = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}_${
    //   script.name
    // }`;
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

export default SetupExperienceScriptCard;
