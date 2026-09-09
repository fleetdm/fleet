import React from "react";

import InfoBanner from "components/InfoBanner";

const baseClass = "apple-bm-token-invalid-message";

const orgNameList = (orgNames: string[]) => {
  if (orgNames.length <= 2) {
    return orgNames.join(" and ");
  }
  return `${orgNames.slice(0, -1).join(", ")}, and ${
    orgNames[orgNames.length - 1]
  }`;
};

interface IAppleBMTokenInvalidMessageProps {
  /** Organization names of the invalid AB tokens */
  orgNames: string[];
}

const AppleBMTokenInvalidMessage = ({
  orgNames,
}: IAppleBMTokenInvalidMessageProps) => {
  const isPlural = orgNames.length > 1;

  return (
    <InfoBanner className={baseClass} color="yellow">
      Your Apple Business (AB) {isPlural ? "tokens" : "token"} for{" "}
      {orgNameList(orgNames)} {isPlural ? "are" : "is"} invalid. macOS, iOS, and
      iPadOS hosts won’t automatically enroll into Fleet. Users with the admin
      role in Fleet can renew the {isPlural ? "tokens" : "token"}.
    </InfoBanner>
  );
};

export default AppleBMTokenInvalidMessage;
