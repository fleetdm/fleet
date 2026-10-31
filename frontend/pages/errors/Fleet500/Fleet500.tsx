import React from "react";

import CustomLink from "components/CustomLink";
import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

const Fleet500 = () => (
  <div className="error-page__details">
    <h1 className="error-page__status-code">500</h1>
    <p className="error-page__subtitle">Oh, something went wrong.</p>
    <p className="error-page__message">
      If you believe this is a bug, please{" "}
      <CustomLink url={GITHUB_NEW_ISSUE_LINK} text="file an issue" newTab />
    </p>
  </div>
);

export default Fleet500;
