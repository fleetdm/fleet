import React from "react";

const baseClass = "profile-list-heading";

const ProfileListHeading = () => {
  return (
    <div className={baseClass}>
      <span>Configuration profile</span>
      <span>Actions</span>
    </div>
  );
};

export default ProfileListHeading;
