import React from "react";
import FileSaver from "file-saver";
import classnames from "classnames";

import { uploadedFromNow } from "utilities/date_format";

import Icon from "components/Icon";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Button from "components/buttons/Button";
import {
  IAppleSetupEnrollmentProfileResponse,
  IDefaultAppleSetupEnrollmentProfileResponse,
} from "services/entities/mdm";

interface IBaseProps<TProfile> {
  profile: TProfile;
}

interface IDefaultProfileProps
  extends IBaseProps<IDefaultAppleSetupEnrollmentProfileResponse> {
  defaultProfile: true;
}

interface ICustomProfileProps
  extends IBaseProps<IAppleSetupEnrollmentProfileResponse> {
  defaultProfile?: false;
  onDelete: () => void;
}

type ISetupAssistantProfileCardProps =
  | IDefaultProfileProps
  | ICustomProfileProps;

const SetupAssistantProfileCard = (props: ISetupAssistantProfileCardProps) => {
  const baseClass = "setup-assistant-profile-card";

  const cardClassName = classnames(baseClass, {
    [`${baseClass}--default-profile`]: props.defaultProfile,
  });

  const onDownload = () => {
    const date = new Date();
    const filename = `${date.toISOString().split("T")[0]}_${
      props.defaultProfile
        ? "default-automatic-enrollment.json"
        : props.profile.name
    }`;
    const file = new global.window.File(
      [JSON.stringify(props.profile.enrollment_profile, null, 2)],
      filename
    );

    FileSaver.saveAs(file);
  };

  return (
    <Card paddingSize="medium" className={cardClassName}>
      <Graphic name="file-configuration-profile" />
      <div className={`${baseClass}__info`}>
        {props.defaultProfile ? (
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
            <span className={`${baseClass}__profile-name`}>
              {props.profile.name}
            </span>
            <span className={`${baseClass}__uploaded-at`}>
              {uploadedFromNow(props.profile.uploaded_at)}
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
        {!props.defaultProfile && (
          <Button
            className={`${baseClass}__delete-button`}
            variant="icon"
            onClick={props.onDelete}
          >
            <Icon name="trash" />
          </Button>
        )}
      </div>
    </Card>
  );
};

export default SetupAssistantProfileCard;
