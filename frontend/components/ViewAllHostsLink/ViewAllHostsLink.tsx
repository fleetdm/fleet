import React from "react";
import PATHS from "router/paths";
import { Link, browserHistory } from "react-router";
import classnames from "classnames";

import Icon from "components/Icon";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

interface IHostLinkProps {
  queryParams?: QueryParams;
  className?: string;
  /** Shows right chevron without text */
  condensed?: boolean;
}

const baseClass = "view-all-hosts-link";

const ViewAllHostsLink = ({
  queryParams,
  className,
  condensed = false,
}: IHostLinkProps): JSX.Element => {
  const viewAllHostsLinkClass = classnames(baseClass, className);

  const path = queryParams
    ? `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams(queryParams)}`
    : PATHS.MANAGE_HOSTS;

  return (
    <Link className={viewAllHostsLinkClass} to={path} title="host-link">
      {!condensed && <span>View all hosts</span>}
      <Icon
        name="chevron"
        className={`${baseClass}__icon`}
        direction="right"
        color="core-fleet-blue"
      />
    </Link>
  );
};
export default ViewAllHostsLink;
