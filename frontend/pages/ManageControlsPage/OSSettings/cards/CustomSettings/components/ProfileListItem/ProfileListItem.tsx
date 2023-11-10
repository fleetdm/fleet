import React from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";

import { IMdmProfile } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ListItem from "components/ListItem";

const baseClass = "profile-list-item";

interface IProfileDetailsProps {
  platform: "darwin" | "windows";
  createdAt: string;
}

const ProfileDetails = ({ platform, createdAt }: IProfileDetailsProps) => {
  const getPlatformName = () => {
    return platform === "darwin" ? "macOS" : "Windows";
  };

  return (
    <div className={`${baseClass}__profile-details`}>
      <span className={`${baseClass}__platform`}>{getPlatformName()}</span>
      <span>&bull;</span>
      <span className={`${baseClass}__list-item-uploaded`}>
        {`Uploaded ${formatDistanceToNow(new Date(createdAt))} ago`}
      </span>
    </div>
  );
};

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
    <ListItem
      className={baseClass}
      graphic="file-configuration-profile"
      title={profile.name}
      details={
        <ProfileDetails
          platform={profile.platform}
          createdAt={profile.created_at}
        />
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
            onClick={() => onDelete(profile)}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </>
      }
    />
  );
};

export default ProfileListItem;
