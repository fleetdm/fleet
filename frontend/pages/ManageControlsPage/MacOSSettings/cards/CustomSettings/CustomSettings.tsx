import React from "react";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "custom-settings";

const CustomSettings = () => {
  const profiles = [1, 2].map((profile) => {
    return (
      <li key={profile} className={`${baseClass}__profile`}>
        <div className={`${baseClass}__profile-data`}>
          <Icon name="profile" />
          <div className={`${baseClass}__profile-info`}>
            <span className={`${baseClass}__profile-name`}>Restrictions</span>
            <span className={`${baseClass}__profile-uploaded`}>
              Uploaded 2 hours ago
            </span>
          </div>
        </div>
        <div className={`${baseClass}__profile-actions`}>
          <Button
            className={`${baseClass}__download-button`}
            variant="text-icon"
            onClick={() => {
              return null;
            }}
          >
            <Icon name="download" />
          </Button>
          <Button
            className={`${baseClass}__delete-button`}
            variant="text-icon"
            onClick={() => {
              return null;
            }}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </div>
      </li>
    );
  });

  return (
    <div className={baseClass}>
      <h2>Custom Settings</h2>
      <p className={`${baseClass}__description`}>
        Create and upload configuration profiles to apply custom settings.{" "}
        <CustomLink
          newTab
          text="Learn how"
          url="https://fleetdm.com/docs/controls#macos-settings"
        />
      </p>

      <div className={`${baseClass}__profiles`}>
        <div className={`${baseClass}__profiles-header`}>
          <span>Configuration profile</span>
          <span>Actions</span>
        </div>
        <ul className={`${baseClass}__profile-list`}>{profiles}</ul>
      </div>

      <div className={`${baseClass}__profile-uploader`}>
        <Icon name="profile" />
        <p>Configuration profile (.mobileconfig)</p>
        <Button
          onClick={() => {
            return null;
          }}
        >
          Upload
        </Button>
      </div>
    </div>
  );
};

export default CustomSettings;
