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
  excludeChevron?: boolean;
  responsive?: boolean;
  customText?: string;
  /** Table links shows on row hover and tab focus only */
  rowHover?: boolean;
  // don't actually create a link, useful when click is handled by an ancestor
  noLink?: boolean;
}

const baseClass = "view-all-hosts-link";

const ViewAllHostsLink = ({
  queryParams,
  className,
  platformLabelId,
  condensed = false,
  excludeChevron = false,
  responsive = false,
  customText,
  rowHover = false,
  noLink = false,
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
    <Link
      className={viewAllHostsLinkClass}
      to={noLink ? "" : path}
      title="host-link"
    >
      {!condensed && (
        <span
          className={`${baseClass}__text${responsive ? "--responsive" : ""}`}
        >
          {customText ?? "View all hosts"}
        </span>
      )}
      {!excludeChevron && (
        <Icon
          name="chevron-right"
          className={`${baseClass}__icon`}
          color="core-fleet-blue"
        />
      )}
    </Link>
  );
};
export default ViewAllHostsLink;
