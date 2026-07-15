import React from "react";

import { format } from "date-fns";
import { timeAgo } from "utilities/date_format";
import FileSaver from "file-saver";
import classnames from "classnames";

import { IMdmProfile, ProfilePlatform } from "interfaces/mdm";
import { isAppleDevice, isIPadOrIPhone } from "interfaces/platform";
import mdmAPI, { isDDMProfile } from "services/entities/mdm";

import Button from "components/buttons/Button";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import strUtils from "utilities/strings";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "profile-list-item";

const LabelCount = ({
  className,
  count,
}: {
  className: string;
  count: number;
}) => (
  <div className={`${className}__labels--count`}>
    <Icon name="filter" color="ui-fleet-black-75" />
    {`${count} ${strUtils.pluralize(count, "label")}`}
  </div>
);

interface IProfileDetailsProps {
  platform: ProfilePlatform;
  uploadedAt: string;
  isDDM?: boolean;
}

const ProfileDetails = ({
  platform,
  uploadedAt,
  isDDM,
}: IProfileDetailsProps) => {
  const getPlatformName = () => {
    switch (platform) {
      case "windows":
        return "Windows";
      case "android":
        return "Android";
      case "linux":
        return "Linux";
      default:
        return isDDM
          ? "macOS, iOS, iPadOS (declaration)"
          : "macOS, iOS, iPadOS";
    }
  };

  return (
    <div className={`${baseClass}__profile-details`}>
      <span className={`${baseClass}__platform`}>{getPlatformName()}</span>
      <span>&bull;</span>
      <span className={`${baseClass}__list-item-uploaded`}>
        {`Uploaded ${timeAgo(new Date(uploadedAt), { addSuffix: true })}`}
      </span>
    </div>
  );
};

const createProfileExtension = (profile: IMdmProfile) => {
  if (isDDMProfile(profile)) {
    return "json";
  }
  if (profile.platform === "android") {
    return "json";
  }
  return isAppleDevice(profile.platform) ? "mobileconfig" : "xml";
};

const createFileContent = async (profile: IMdmProfile) => {
  const content = await mdmAPI.downloadProfile(profile.profile_uuid);
  if (isDDMProfile(profile)) {
    return JSON.stringify(content, null, 2);
  }
  if (profile.platform === "android") {
    return JSON.stringify(content, null, 2);
  }
  return content;
};

interface IProfileListItemProps {
  isPremium: boolean;
  profile: IMdmProfile;
  onClickInfo: (profile: IMdmProfile) => void;
  onClickDelete: (profile: IMdmProfile) => void;
  setProfileLabelsModalData: React.Dispatch<
    React.SetStateAction<IMdmProfile | null>
  >;
  isTechnician?: boolean;
}

const ProfileListItem = ({
  isPremium,
  profile,
  onClickInfo,
  onClickDelete,
  setProfileLabelsModalData,
  isTechnician,
}: IProfileListItemProps) => {
  const {
    updated_at,
    labels_include_all,
    labels_include_any,
    labels_exclude_any,
    name,
    platform,
    scope,
  } = profile;
  const subClass = "list-item";

  // iOS/iPadOS don't support user channels, so never show the user-scope icon
  // for them (matches the host details OS settings table).
  const isUserScoped = scope === "User" && !isIPadOrIPhone(platform);

  const onClickDownload = async () => {
    const fileContent = await createFileContent(profile);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const extension = createProfileExtension(profile);
    const filename = `${formatDate}_${name}.${extension}`;
    const file = new File([fileContent], filename);
    FileSaver.saveAs(file);
  };

  const labels = [
    ...(labels_include_all ?? []),
    ...(labels_include_any ?? []),
    ...(labels_exclude_any ?? []),
  ];

  const renderLabelInfo = () => {
    if (!isPremium || labels.length === 0) {
      return null;
    }

    return (
      <div className={`${subClass}__labels`}>
        {labels.some((label) => label.broken) && <Icon name="warning" />}
        <LabelCount className={subClass} count={labels.length} />
      </div>
    );
  };

  return (
    // TODO - refactor to use ListItem
    <div className={classnames(subClass, baseClass)}>
      <div className={`${subClass}__main-content`}>
        <Graphic name="file-configuration-profile" />
        <div className={`${subClass}__info`}>
          <div className={`${baseClass}__title-row`}>
            <TooltipWrapper
              tipContent={`UUID: ${profile.profile_uuid}`}
              underline={false}
              position="top"
              showArrow
            >
              <span className={`${subClass}__title`}>{name}</span>
            </TooltipWrapper>
            {isUserScoped && (
              <TooltipWrapper
                className={`${baseClass}__scope-tooltip`}
                tipContent="Scoped to the user channel."
                underline={false}
                position="top"
                showArrow
              >
                <Icon name="user" />
              </TooltipWrapper>
            )}
          </div>
          <div className={`${subClass}__details`}>
            <ProfileDetails
              platform={platform}
              uploadedAt={updated_at}
              isDDM={isDDMProfile(profile)}
            />
          </div>
        </div>
      </div>
      <div className={`${subClass}__actions-wrap`}>
        {renderLabelInfo()}
        <div className={`${subClass}__actions`}>
          <Button
            className={`${subClass}__action-button`}
            variant="icon"
            onClick={() => onClickInfo(profile)}
          >
            <Icon name="info" size="medium" />
          </Button>
          {isPremium && labels.length > 0 && (
            <Button
              className={`${subClass}__action-button`}
              variant="icon"
              onClick={() => setProfileLabelsModalData({ ...profile })}
            >
              <Icon name="filter" />
            </Button>
          )}
          <Button
            className={`${subClass}__action-button`}
            variant="icon"
            onClick={onClickDownload}
          >
            <Icon name="download" />
          </Button>
          {!isTechnician && (
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Button
                  disabled={disableChildren}
                  className={`${subClass}__action-button`}
                  variant="icon"
                  onClick={() => onClickDelete(profile)}
                >
                  <Icon name="trash" />
                </Button>
              )}
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default ProfileListItem;
