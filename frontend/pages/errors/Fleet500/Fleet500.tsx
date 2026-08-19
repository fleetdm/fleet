import React from "react";

import { GITHUB_NEW_ISSUE_LINK } from "utilities/constants";

const baseClass = "fleet-500";

const Fleet500 = () => (
  <div className="error-page__details">
    <h1 className="error-page__status-code">500</h1>
    <p className="error-page__subtitle">Oh, something went wrong.</p>
    <p className="error-page__message">
      Please{" "}
      <a
        className={`${baseClass}__link`}
        href={GITHUB_NEW_ISSUE_LINK}
        target="_blank"
        rel="noopener noreferrer"
      >
        file an issue
      </a>{" "}
      if you believe this is a bug.
    </p>
  </div>
);

export default Fleet500;
