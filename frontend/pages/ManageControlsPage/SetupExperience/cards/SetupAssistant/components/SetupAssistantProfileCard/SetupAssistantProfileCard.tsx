import React from "react";
import FileSaver from "file-saver";

import { uploadedFromNow } from "utilities/date_format";

import Icon from "components/Icon";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Button from "components/buttons/Button";
import { IAppleSetupEnrollmentProfileResponse } from "services/entities/mdm";

const baseClass = "setup-assistant-profile-card";
interface ISetupAssistantProfileCardProps {
  profile: IAppleSetupEnrollmentProfileResponse;
  onDelete: () => void;
}

const SetupAssistantProfileCard = ({
  profile,
  onDelete,
}: ISetupAssistantProfileCardProps) => {
  const onDownload = () => {
    const date = new Date();
    const filename = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}_${
      profile.name
    }`;
    const file = new global.window.File(
      [JSON.stringify(profile.enrollment_profile)],
      filename
    );

    FileSaver.saveAs(file);
  };

  return (
    <Card paddingSize="medium" className={baseClass}>
      <Graphic name="file-configuration-profile" />
      <div className={`${baseClass}__info`}>
        <span className={`${baseClass}__profile-name`}>{profile.name}</span>
        <span className={`${baseClass}__uploaded-at`}>
          {uploadedFromNow(profile.uploaded_at)}
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

export default SetupAssistantProfileCard;
