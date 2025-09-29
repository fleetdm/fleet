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
import { isSafeImagePreviewUrl } from "pages/SoftwarePage/helpers";
import TooltipWrapperArchLinuxRolling from "components/TooltipWrapperArchLinuxRolling";

import SoftwareIcon from "../../icons/SoftwareIcon";
import OSIcon from "../../icons/OSIcon";

const baseClass = "software-details-summary";

interface ISoftwareDetailsSummaryProps {
  title: string;
  type?: string;
  hosts: number;
  countsUpdatedAt?: string;
  /** The query param that will be added when user clicks on the host count
   * Optional as isPreview mode doesn't have host count/link
   */
  queryParams?: QueryParams;
  name?: string;
  source?: string;
  versions?: number;
  iconUrl?: string | null;
  /** Displays OS icon instead of Software icon */
  isOperatingSystem?: boolean;
  /** Displays an edit CTA to edit the software's icon
   * Should only be defined for team view of an installable software */
  onClickEditIcon?: () => void;
  /** undefined unless previewing icon, in which case is string or null */
  iconPreviewUrl?: string | null;
  /** timestamp of when icon was last uploaded, used to force refresh of cached icon */
  iconUploadedAt?: string;
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
  iconPreviewUrl,
  iconUploadedAt,
}: ISoftwareDetailsSummaryProps) => {
  const hostCountPath = getPathWithQueryParams(paths.MANAGE_HOSTS, queryParams);

  // Remove host count for tgz_packages only and if viewing details summary from edit icon preview modal
  const showHostCount =
    source !== "tgz_packages" && iconPreviewUrl === undefined;

  const renderSoftwareIcon = () => {
    if (
      typeof iconPreviewUrl === "string" &&
      isSafeImagePreviewUrl(iconPreviewUrl)
    ) {
      return (
        <img
          src={iconPreviewUrl}
          alt="Uploaded icon preview"
          style={{ width: 96, height: 96 }}
        />
      );
    }

    return (
      <SoftwareIcon
        name={name}
        source={source}
        url={iconUrl}
        uploadedAt={iconUploadedAt}
        size="xlarge"
      />
    );
  };

  return (
    <>
      <div className={baseClass}>
        {isOperatingSystem ? (
          <OSIcon name={name} size="xlarge" />
        ) : (
          renderSoftwareIcon()
        )}
        <dl className={`${baseClass}__info`}>
          <h1>
            {title === "Arch Linux rolling" ||
            title === "Arch Linux ARM rolling" ||
            title === "Manjaro Linux rolling" ||
            title === "Manjaro Linux ARM rolling" ? (
              <span>
                {title.slice(0, -7 /* removing lowercase rolling suffix */)}
                <TooltipWrapperArchLinuxRolling />
              </span>
            ) : (
              title
            )}
            {onClickEditIcon && (
              <div className={`${baseClass}__edit-icon`}>
                <Button
                  onClick={onClickEditIcon}
                  className={`${baseClass}__edit-icon-btn`}
                  variant="icon"
                >
                  <Icon name="pencil" />
                </Button>
              </div>
            )}
          </h1>
          <dl className={`${baseClass}__description-list`}>
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
