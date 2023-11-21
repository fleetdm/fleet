import React from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { IMdmProfile } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "profile-list-item";

interface IProfileListItemProps {
  profile: IMdmProfile;
  onDelete: (profile: IMdmProfile) => void;
}

const ProfileListItem = ({ profile, onDelete }: IProfileListItemProps) => {
  const onClickDownload = async () => {
    const fileContent = await mdmAPI.downloadProfile(profile.profile_id);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const filename = `${formatDate}_${profile.name}.mobileconfig`;
    const file = new File([fileContent], filename);
    FileSaver.saveAs(file);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__list-item-data`}>
        <Icon name="profile" />
        <div className={`${baseClass}__list-item-info`}>
          <span className={`${baseClass}__list-item-name`}>{profile.name}</span>
          <span className={`${baseClass}__list-item-uploaded`}>
            {`Uploaded ${formatDistanceToNow(
              new Date(profile.created_at)
            )} ago`}
          </span>
        </div>
      </div>
      <div className={`${baseClass}__list-item-actions`}>
        <Button
          className={`${baseClass}__list-item-button`}
          variant="text-icon"
          onClick={onClickDownload}
        >
          <Icon name="download" />
        </Button>
        <Button
          className={`${baseClass}__list-item-button`}
          variant="text-icon"
          onClick={() => onDelete(profile)}
        >
          <Icon name="trash" color="ui-fleet-black-75" />
        </Button>
      </div>
    </div>
  );
};

export default ProfileListItem;
