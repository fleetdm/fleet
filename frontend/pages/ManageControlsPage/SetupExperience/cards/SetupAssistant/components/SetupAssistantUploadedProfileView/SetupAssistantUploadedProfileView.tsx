import React, { useState } from "react";

import URL_PREFIX from "router/url_prefix";
import { uploadedFromNow } from "utilities/date_format";
import endpoints from "utilities/endpoints";

import CustomLink from "components/CustomLink";
import Icon from "components/Icon";
import Card from "components/Card";
import Graphic from "components/Graphic";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "setup-assistant-uploaded-profile-view";

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
        className={`${baseClass}__download-button`}
      >
        <Icon name="download" />
      </Button>
    </form>
  );
};

const AdvancedOptionsForm = () => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [releaseDevice, setReleaseDevice] = useState(false);

  const accordionText = showAdvancedOptions ? "Hide" : "Show";
  const icon = showAdvancedOptions ? "chevron-up" : "chevron-down";

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    console.log("release?", releaseDevice);
  };

  const tooltip = (
    <>
      When enabled, you&apos;re responsible for sending the DeviceConfigured
      command. (Default: <b>Off</b>)
    </>
  );

  return (
    <div className={`${baseClass}__advanced-options`}>
      <div
        className={`${baseClass}__accordion-title`}
        onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
      >
        <span>{accordionText} advanced options</span>
        <Icon name={icon} color="core-fleet-blue" />
      </div>
      {showAdvancedOptions && (
        <form onSubmit={handleSubmit}>
          <Checkbox
            value={releaseDevice}
            onChange={() => setReleaseDevice(!releaseDevice)}
          >
            <TooltipWrapper tipContent={tooltip}>
              Release device manually
            </TooltipWrapper>
          </Checkbox>
          <Button type="submit">Save</Button>
        </form>
      )}
    </div>
  );
};

interface ISetupAssistantUploadedProfileViewProps {
  profileMetaData: any;
  currentTeamId: number;
  onDelete: (packageMetaData: any) => void;
}

const SetuAssistantUploadedProfileView = ({
  profileMetaData,
  currentTeamId,
  onDelete,
}: ISetupAssistantUploadedProfileViewProps) => {
  profileMetaData = {
    title: "test-profile.json",
    created_at: "2021-08-25T20:00:00Z",
    token: "123-abc",
  };

  const { origin } = global.window.location;
  const path = `${endpoints.MDM_BOOTSTRAP_PACKAGE}`;
  const url = `${origin}${URL_PREFIX}/api${path}`;

  return (
    <div className={baseClass}>
      <p>
        Add an automatic enrollment profile to customize the macOS Setup
        Assistant.
        <CustomLink
          url=" https://fleetdm.com/learn-more-about/setup-assistant"
          text="Learn how"
          newTab
        />
      </p>
      <Card paddingSize="medium" className={`${baseClass}__uploaded-profile`}>
        <Graphic name="file-configuration-profile" />
        <div className={`${baseClass}__info`}>
          <span className={`${baseClass}__profile-name`}>
            {profileMetaData.title}
          </span>
          <span className={`${baseClass}__uploaded-at`}>
            {uploadedFromNow(profileMetaData.created_at)}
          </span>
        </div>
        <div className={`${baseClass}__actions`}>
          <DownloadPackageButton
            className={`${baseClass}__download-package`}
            url={url}
            token={profileMetaData.token}
          />
          <Button
            className={`${baseClass}__delete-button`}
            variant="text-icon"
            onClick={() => onDelete(profileMetaData)}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </div>
      </Card>

      <AdvancedOptionsForm />
    </div>
  );
};

export default SetuAssistantUploadedProfileView;
