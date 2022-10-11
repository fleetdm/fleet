import React from "react";

import PhoneHome from "../../../../../assets/images/phone-home.svg";

const baseClass = "awaiting-results";

const AwaitingResults = () => {
  return (
    <div className={baseClass}>
      <img src={PhoneHome} alt="awaiting results" />
      <span className={`${baseClass}__title`}>Phoning home...</span>
      <p className={`${baseClass}__description`}>
        There are currently no results to your query. Please wait while we talk
        to more hosts.
      </p>
    </div>
  );
};

export default AwaitingResults;
