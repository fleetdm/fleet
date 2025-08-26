/**
software/titles/:id > Top section
software/versions/:id > Top section
software/os/:id > Top section
*/

import React from "react";

import { getPathWithQueryParams, QueryParams } from "utilities/url";
import paths from "router/paths";

import DataSet from "components/DataSet";
import LastUpdatedHostCount from "components/LastUpdatedHostCount";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import SoftwareIcon from "../../icons/SoftwareIcon";
import OSIcon from "../../icons/OSIcon";

/** This class are re-used on a edit icon modal > preview */
export const baseClass = "software-details-summary";
/** This class are re-used on a edit icon modal > preview */
export const infoClass = `${baseClass}__info`;
/** This class are re-used on a edit icon modal > preview */
export const descriptionListClass = `${baseClass}__description-list`;

interface ISoftwareDetailsSummaryProps {
  title: string;
  type?: string;
  hosts: number;
  countsUpdatedAt?: string;
  /** The query param that will be added when user clicks on the host count */
  queryParams: QueryParams;
  name?: string;
  source?: string;
  versions?: number;
  iconUrl?: string;
  /** Displays OS icon instead of Software icon */
  isOperatingSystem?: boolean;
  onClickEditIcon?: () => void;
}

const SoftwareDetailsSummary = ({
  title,
  type,
  hosts,
  countsUpdatedAt,
  queryParams,
  name,
  source,
  versions,
  iconUrl,
  isOperatingSystem,
  onClickEditIcon,
}: ISoftwareDetailsSummaryProps) => {
  const hostCountPath = getPathWithQueryParams(paths.MANAGE_HOSTS, queryParams);
  // Remove host count for tgz_packages only
  const showHostCount = source !== "tgz_packages";

  return (
    <>
      <div className={baseClass}>
        {isOperatingSystem ? (
          <OSIcon name={name} size="xlarge" />
        ) : (
          <SoftwareIcon
            name={name}
            source={source}
            url={iconUrl}
            size="xlarge"
          />
        )}
        <dl className={infoClass}>
          <h1>
            {title}
            {onClickEditIcon && (
              <div className={`${baseClass}__edit-icon`}>
                <GitOpsModeTooltipWrapper
                  position="right"
                  tipOffset={8}
                  renderChildren={(disableChildren) => (
                    <Button
                      disabled={disableChildren}
                      onClick={onClickEditIcon}
                      className={`${baseClass}__edit-icon-btn`}
                      variant="text-icon"
                    >
                      <Icon name="pencil" />
                    </Button>
                  )}
                />
              </div>
            )}
          </h1>
          <dl className={descriptionListClass}>
            {!!type && <DataSet title="Type" value={type} />}

            {!!versions && <DataSet title="Versions" value={versions} />}
            {showHostCount && (
              <DataSet
                title="Hosts"
                value={
                  <LastUpdatedHostCount
                    hostCount={
                      <TooltipWrapper tipContent="View all hosts">
                        <CustomLink
                          url={hostCountPath}
                          text={hosts.toString()}
                        />
                      </TooltipWrapper>
                    }
                    lastUpdatedAt={countsUpdatedAt}
                  />
                }
              />
            )}
          </dl>
        </dl>
      </div>
    </>
  );
};

export default SoftwareDetailsSummary;
