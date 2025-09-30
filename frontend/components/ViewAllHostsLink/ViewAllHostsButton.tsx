import React from "react";
import PATHS from "router/paths";
import { browserHistory } from "react-router";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { getPathWithQueryParams, QueryParams } from "utilities/url";

interface IHostLinkProps {
  queryParams?: QueryParams;
  className?: string;
  /** Including the platformId will view all hosts for the platform provided */
  platformLabelId?: number;
  /** Shows right chevron without text */
  condensed?: boolean;
  excludeChevron?: boolean;
  responsive?: boolean;
  customContent?: React.ReactNode;
  /** Table links shows on row hover and tab focus only */
  rowHover?: boolean;
  /** Don't actually create a button, useful when click is handled by an ancestor */
  noLink?: boolean;
}

const baseClass = "view-all-hosts-button";

const ViewAllHostsButton = ({
  queryParams,
  className,
  platformLabelId,
  condensed = false,
  excludeChevron = false,
  responsive = false,
  customContent,
  rowHover = false,
  noLink = false,
}: IHostLinkProps): JSX.Element => {
  const viewAllHostsButtonClass = classnames(baseClass, className, {
    [`${baseClass}__condensed`]: condensed,
    "row-hover-button": rowHover,
  });

  const endpoint = platformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(platformLabelId)
    : PATHS.MANAGE_HOSTS;

  const path = getPathWithQueryParams(endpoint, queryParams);

  const onClick = (e: MouseEvent): void => {
    if (!noLink) {
      e.stopPropagation(); // Allows for button to have different onClick behavior than the row's onClick behavior

      if (path) {
        browserHistory.push(path);
      }
    }
  };

  return (
    <Button
      className={viewAllHostsButtonClass}
      onClick={onClick}
      variant="inverse"
      size="small"
    >
      {!condensed && (
        <span
          className={`${baseClass}__text${responsive ? "--responsive" : ""}`}
        >
          {customContent ?? "View all hosts"}
        </span>
      )}
      {!excludeChevron && (
        <Icon
          name="chevron-right"
          className={`${baseClass}__icon`}
          color="ui-fleet-black-75"
        />
      )}
    </Button>
  );
};

export default ViewAllHostsButton;
