import React from "react";
import PATHS from "router/paths";
import { browserHistory } from "react-router";
import classnames from "classnames";

import { IDropdownOption } from "interfaces/dropdownOption";

import Button from "components/buttons/Button";
import ActionsDropdown from "components/ActionsDropdown";
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
  /** Custom text replaces "View all hosts" in button or "Actions" in dropdown */
  customText?: string;
  /** Table links shows on row hover and tab focus only */
  rowHover?: boolean;
  /** Don't actually create a button, useful when click is handled by an ancestor */
  noLink?: boolean;
  /** When provided, replaces View all hosts button with ActionDropdown */
  dropdown?: { options: IDropdownOption[]; onChange: (value: string) => void };
}

const baseClass = "view-all-hosts-button";

const ViewAllHostsButton = ({
  queryParams,
  className,
  platformLabelId,
  condensed = false,
  excludeChevron = false,
  responsive = false,
  customText,
  rowHover = false,
  noLink = false,
  dropdown,
}: IHostLinkProps): JSX.Element => {
  console.log("rowHover", rowHover);
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

  if (dropdown) {
    return (
      <ActionsDropdown
        className={viewAllHostsButtonClass}
        options={dropdown.options}
        onChange={dropdown.onChange}
        placeholder={customText || "Actions"}
        variant="small-button"
        menuAlign="right"
      />
    );
  }

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
          {customText ?? "View all hosts"}
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
