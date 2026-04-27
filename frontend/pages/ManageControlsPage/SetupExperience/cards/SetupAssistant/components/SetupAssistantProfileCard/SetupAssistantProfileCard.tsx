import React from "react";
import FileSaver from "file-saver";

import { uploadedFromNow } from "utilities/date_format";

import Icon from "components/Icon";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Button from "components/buttons/Button";
import { IAppleSetupEnrollmentProfileResponse } from "services/entities/mdm";

interface ISetupAssistantProfileCardProps {
  profile: IAppleSetupEnrollmentProfileResponse;
  onDelete?: () => void;
  defaultProfile?: boolean;
}

const SetupAssistantProfileCard = ({
  profile,
  onDelete,
  defaultProfile = false,
}: ISetupAssistantProfileCardProps) => {
  const baseClass = `setup-assistant-profile-card${
    defaultProfile ? "-default-profile" : ""
  }`;
  const onDownload = () => {
    const date = new Date();
    const filename = `${date.toISOString().split("T")[0]}_${
      defaultProfile ? "default-automatic-enrollment.json" : profile.name
    }`;
    const file = new global.window.File(
      [JSON.stringify(profile.enrollment_profile, null, 2)],
      filename
    );

    FileSaver.saveAs(file);
  };

  return (
    <Card paddingSize="medium" className={baseClass}>
      <Graphic name="file-configuration-profile" />
      <div className={`${baseClass}__info`}>
        {defaultProfile ? (
          <>
            <span className={`${baseClass}__profile-name`}>
              Default profile
            </span>
            <span className={`${baseClass}__description`}>
              Hosts use this profile, unless you add your own.
            </span>
          </>
        ) : (
          <>
            <span className={`${baseClass}__profile-name`}>{profile.name}</span>
            <span className={`${baseClass}__uploaded-at`}>
              {uploadedFromNow(profile.uploaded_at)}
            </span>
          </>
        )}
      </div>
      <div className={`${baseClass}__actions`}>
        <Button
          className={`${baseClass}__download-button`}
          variant="icon"
          onClick={onDownload}
        >
          <Icon name="download" />
        </Button>
        {!defaultProfile && (
          <Button
            className={`${baseClass}__delete-button`}
            variant="icon"
            onClick={onDelete}
          >
            <Icon name="trash" />
          </Button>
        )}
      </div>
    </Card>
  );
};

export default SetupAssistantProfileCard;
