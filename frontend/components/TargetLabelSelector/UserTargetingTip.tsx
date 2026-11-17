import React from "react";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

// Label copy says "hosts," so admins don't expect labels to target people.
const UserTargetingTip = () => (
  <span className="form-field__help-text">
    To target users, use labels based on IdP groups or departments.{" "}
    <CustomLink
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/target-users-with-labels`}
      text="Learn more"
      newTab
    />
  </span>
);

export default UserTargetingTip;
