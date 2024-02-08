import React from "react";
import PATHS from "router/paths";
import { Link } from "react-router";
import classnames from "classnames";

import Icon from "components/Icon";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

interface IHostLinkProps {
  queryParams?: QueryParams;
  className?: string;
  /** Including the platformId will view all hosts for the platform provided */
  platformLabelId?: number;
  /** Shows right chevron without text */
  condensed?: boolean;
  responsive?: boolean;
  customText?: string;
  /** Table links shows on row hover only */
  rowHover?: boolean;
}

const baseClass = "view-all-hosts-link";

const ViewAllHostsLink = ({
  queryParams,
  className,
  platformLabelId,
  condensed = false,
  responsive = false,
  customText,
  rowHover = false,
}: IHostLinkProps): JSX.Element => {
  const viewAllHostsLinkClass = classnames(baseClass, className, {
    "row-hover-link": rowHover,
  });

  const endpoint = platformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(platformLabelId)
    : PATHS.MANAGE_HOSTS;

  const path = queryParams
    ? `${endpoint}?${buildQueryStringFromParams(queryParams)}`
    : endpoint;

  return (
    <Link className={viewAllHostsLinkClass} to={path} title="host-link">
      {!condensed && (
        <span
          className={`${baseClass}__text${responsive ? "--responsive" : ""}`}
        >
          {customText ?? "View all hosts"}
        </span>
      )}
      <Icon
        name="chevron-right"
        className={`${baseClass}__icon`}
        color="core-fleet-blue"
      />
    </Link>
  );
};
export default ViewAllHostsLink;
