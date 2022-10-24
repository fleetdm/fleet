import React from "react";
import PATHS from "router/paths";
import classnames from "classnames";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import { browserHistory } from "react-router";

interface IHostLinkProps {
  queryParams?: QueryParams;
  className?: string;
  condensed?: boolean;
}

const baseClass = "view-all-hosts-link";

const ViewAllHostsLink = ({
  queryParams,
  className,
  condensed = false,
}: IHostLinkProps): JSX.Element => {
  const linkClasses = classnames(baseClass, className);

  const onClick = (): void => {
    const path = queryParams
      ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(queryParams)}`
      : PATHS.MANAGE_HOSTS;

    browserHistory.push(path);
  };

  return (
    <Button className={linkClasses} onClick={onClick} variant="text-icon">
      <>
        {condensed ? "" : "View all hosts"}
        <Icon
          name="chevron"
          className={`${baseClass}__link-icon`}
          direction="right"
          color="#6a67fe"
        />
      </>
    </Button>
  );
};
export default ViewAllHostsLink;
