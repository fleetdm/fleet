import React from "react";
import { formatDistanceToNow } from "date-fns";
import URL_PREFIX from "router/url_prefix";

import { IBootstrapPackageMetadata } from "interfaces/mdm";
import endpoints from "utilities/endpoints";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import Graphic from "components/Graphic";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "bootstrap-package-list-item";

interface IBootstrapPackageListItemProps {
  bootstrapPackage: IBootstrapPackageMetadata;
  onDelete: (bootstrapPackage: IBootstrapPackageMetadata) => void;
}

interface ITestFormProps {
  url: string;
  token: string;
  className?: string;
}

/**
 * This component abstracts away the downloading of the package. It implements this
 * with a browser form that calls the correct url to initiate the package download.
 * We do it this way as this allows us to take advantage of the browsers native
 * downloading UI instead of having to handle this in the Fleet UI.
 * TODO: make common component and use here and in DownloadInstallers.tsx.
 */
const DownloadPackageButton = ({ url, token, className }: ITestFormProps) => {
  return (
    <form
      key="form"
      method="GET"
      action={url}
      target="_self"
      className={className}
    >
      <input type="hidden" name="token" value={token || ""} />
      <Button
        variant="text-icon"
        type="submit"
        className={`${baseClass}__list-item-button`}
      >
        <Icon name="download" />
      </Button>
    </form>
  );
};

const BootstrapPackageListItem = ({
  bootstrapPackage,
  onDelete,
}: IBootstrapPackageListItemProps) => {
  const { origin } = global.window.location;
  const path = `${endpoints.MDM_BOOTSTRAP_PACKAGE}`;
  const url = `${origin}${URL_PREFIX}/api${path}`;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__value-group ${baseClass}__list-item-data`}>
        <Graphic name="file-pkg" />
        <div className={`${baseClass}__list-item-info`}>
          <span className={`${baseClass}__list-item-name`}>
            {bootstrapPackage.name}
          </span>
          <span className={`${baseClass}__list-item-uploaded`}>
            {`Uploaded ${formatDistanceToNow(
              new Date(bootstrapPackage.created_at)
            )} ago`}
          </span>
        </div>
      </div>

      <div
        className={`${baseClass}__value-group ${baseClass}__list-item-actions`}
      >
        <DownloadPackageButton
          className={`${baseClass}__list-item-button`}
          url={url}
          token={bootstrapPackage.token}
        />
        <GitOpsModeTooltipWrapper
          renderChildren={(disabled) => (
            <Button
              className={`${baseClass}__list-item-button`}
              variant="text-icon"
              disabled={disabled}
              onClick={() => onDelete(bootstrapPackage)}
            >
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          )}
        />
      </div>
    </div>
  );
};

export default BootstrapPackageListItem;
