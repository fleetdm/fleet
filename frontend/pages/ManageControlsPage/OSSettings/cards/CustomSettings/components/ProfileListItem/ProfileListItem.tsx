import React from "react";
import { format, formatDistanceToNow } from "date-fns";
import FileSaver from "file-saver";
import classnames from "classnames";

import { IMdmProfile } from "interfaces/mdm";
import mdmAPI, { isDDMProfile } from "services/entities/mdm";

import Button from "components/buttons/Button";
import Graphic from "components/Graphic";
import Icon from "components/Icon";

import strUtils from "utilities/strings";

const baseClass = "profile-list-item";

const LabelCount = ({
  className,
  count,
}: {
  className: string;
  count: number;
}) => (
  <div className={`${className}__labels--count`}>
    {`${count} ${strUtils.pluralize(count, "label")}`}
  </div>
);

interface IProfileDetailsProps {
  platform: string;
  createdAt: string;
  isDDM?: boolean;
}

const ProfileDetails = ({
  platform,
  createdAt,
  isDDM,
}: IProfileDetailsProps) => {
  const getPlatformName = () => {
    if (platform === "windows") return "Windows";
    return isDDM ? "macOS (declaration)" : "macOS";
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

const createProfileExtension = (profile: IMdmProfile) => {
  if (isDDMProfile(profile)) {
    return "json";
  }
  return profile.platform === "darwin" ? "mobileconfig" : "xml";
};

const createFileContent = async (profile: IMdmProfile) => {
  const content = await mdmAPI.downloadProfile(profile.profile_uuid);
  if (isDDMProfile(profile)) {
    return JSON.stringify(content, null, 2);
  }
  return content;
};

interface IProfileListItemProps {
  isPremium: boolean;
  profile: IMdmProfile;
  onDelete: (profile: IMdmProfile) => void;
  setProfileLabelsModalData: React.Dispatch<
    React.SetStateAction<IMdmProfile | null>
  >;
}

const ProfileListItem = ({
  isPremium,
  profile,
  onDelete,
  setProfileLabelsModalData,
}: IProfileListItemProps) => {
  const { created_at, labels, name, platform } = profile;
  const subClass = "list-item";

  const onClickDownload = async () => {
    const fileContent = await createFileContent(profile);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const extension = createProfileExtension(profile);
    const filename = `${formatDate}_${name}.${extension}`;
    const file = new File([fileContent], filename);
    FileSaver.saveAs(file);
  };

  return (
    <div className={classnames(subClass, baseClass)}>
      <div className={`${subClass}__main-content`}>
        <Graphic name="file-configuration-profile" />
        <div className={`${subClass}__info`}>
          <span className={`${subClass}__title`}>{name}</span>
          <div className={`${subClass}__details`}>
            <ProfileDetails
              platform={platform}
              createdAt={created_at}
              isDDM={isDDMProfile(profile)}
            />
          </div>
        </div>
      </div>
      <div className={`${subClass}__actions-wrap`}>
        {isPremium && !!labels?.length && (
          <div className={`${subClass}__labels`}>
            {labels?.some((l) => l.broken) && <Icon name="warning" />}
            <LabelCount className={subClass} count={labels.length} />
          </div>
        )}
        <div className={`${subClass}__actions`}>
          {isPremium && !!labels?.length && (
            <Button
              className={`${subClass}__action-button`}
              variant="text-icon"
              onClick={() => setProfileLabelsModalData({ ...profile })}
            >
              <Icon name="filter" />
            </Button>
          )}
          <Button
            className={`${subClass}__action-button`}
            variant="text-icon"
            onClick={onClickDownload}
          >
            <Icon name="download" />
          </Button>
          <Button
            className={`${subClass}__action-button`}
            variant="text-icon"
            onClick={() => onDelete(profile)}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </div>
      </div>
    </div>
  );
};

export default ProfileListItem;
