import React from "react";

import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

// Label copy says "hosts," so admins don't expect labels to target people or rings.
const UserTargetingTip = () => (
  <span className="form-field__help-text">
    Use labels to target users (IdP groups) or roll out changes in stages.{" "}
    <CustomLink
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/deployment-rings`}
      text="Learn more"
      newTab
    />
  </span>
);

export default UserTargetingTip;
