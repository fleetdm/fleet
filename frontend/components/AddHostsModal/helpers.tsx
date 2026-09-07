import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import React from "react";
import { Link } from "react-router";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

// eslint-disable-next-line import/prefer-default-export
export const renderAppleManualEnrollmentDisabled = (type: string) => {
  return (
    <EmptyState
      className="blocked-manual-enrollment"
      header="Manual enrollment is disabled"
      width="small"
      info={`${type} enrollment is only available through Apple Business (ADE).`}
      primaryButton={
        <Link to={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-abm`} target="_blank">
          <Button>Learn more</Button>
        </Link>
      }
    />
  );
};
