/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React, { useState } from "react";
import classnames from "classnames";

import { stringToClipboard } from "utilities/copy_text";
import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";

const baseClass = "installer-details-widget";

const ANDROID_PLAY_STORE_URL = "https://play.google.com/store/apps/details";

interface IInstallerNameProps {
  name: string;
}

const InstallerName = ({ name }: IInstallerNameProps) => {
  const titleRef = React.useRef<HTMLDivElement>(null);
  const isTruncated = useCheckTruncatedElement(titleRef);

  return (
    <TooltipWrapper
      tipContent={name}
      position="top"
      underline={false}
      disableTooltip={!isTruncated}
      showArrow
    >
      <div ref={titleRef} className={`${baseClass}__title`}>
        {name}
      </div>
    </TooltipWrapper>
  );
};

const renderInstallerDisplayText = (
  installerType: string,
  isFma: boolean,
  androidPlayStoreLink?: string
) => {
  if (installerType === "package") {
    return isFma ? "Fleet-maintained" : "Custom package";
  }
  if (androidPlayStoreLink) {
    return "Google Play Store";
  }
  return "App Store (VPP)";
};

interface IInstallerDetailsWidgetProps {
  className?: string;
  softwareName: string;
  installerType: "package" | "app-store";
  addedTimestamp?: string;
  version?: string | null;
  sha256?: string | null;
  isFma: boolean;
  isScriptPackage: boolean;
  androidPlayStoreLink?: string;
}

const InstallerDetailsWidget = ({
  className,
  softwareName,
  installerType,
  addedTimestamp,
  sha256,
  version,
  isFma,
  isScriptPackage,
  androidPlayStoreLink: androidPlayStoreId,
}: IInstallerDetailsWidgetProps) => {
  const classNames = classnames(baseClass, className);

  const [copyMessage, setCopyMessage] = useState("");

  const onCopySha256 = (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(sha256)
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    setTimeout(() => setCopyMessage(""), 1000);

    return false;
  };

  const renderIcon = () => {
    if (installerType === "app-store") {
      if (androidPlayStoreId) {
        return <SoftwareIcon name="androidPlayStore" size="medium" />;
      }
      return <SoftwareIcon name="appleAppStore" size="medium" />;
    }
    return <Graphic name="file-pkg" />;
  };

  const renderDetails = () => {
    const renderVersionInfo = () => {
      if (isScriptPackage) {
        return null;
      }

      let versionInfo = <span>{version}</span>;

      if (installerType === "app-store") {
        versionInfo = (
          <TooltipWrapper tipContent={<span>Updated every hour.</span>}>
            <span>{version}</span>
          </TooltipWrapper>
        );
      }

      if (!version) {
        versionInfo = (
          <TooltipWrapper
            tipContent={
              <span>
                Fleet couldn&apos;t read the version from {softwareName}.
                {installerType === "package" && (
                  <>
                    {" "}
                    <CustomLink
                      newTab
                      url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
                      text="Learn more"
                      variant="tooltip-link"
                    />
                  </>
                )}
              </span>
            }
          >
            <span>Version (unknown)</span>
          </TooltipWrapper>
        );
      }

      if (androidPlayStoreId) {
        versionInfo = (
          <TooltipWrapper
            tipContent={
              <span>
                See latest version on the{" "}
                <CustomLink
                  text="Play Store"
                  url={getPathWithQueryParams(ANDROID_PLAY_STORE_URL, {
                    id: androidPlayStoreId,
                  })}
                  newTab
                />
              </span>
            }
          >
            <span>Latest</span>
          </TooltipWrapper>
        );
      }

      return <> &bull; {versionInfo}</>;
    };

    const renderTimeStamp = () =>
      addedTimestamp ? (
        <>
          {" "}
          &bull;{" "}
          <TooltipWrapper
            tipContent={internationalTimeFormat(new Date(addedTimestamp))}
            underline={false}
          >
            {addedFromNow(addedTimestamp)}
          </TooltipWrapper>
        </>
      ) : (
        ""
      );

    const renderSha256 = () => {
      return sha256 ? (
        <>
          {" "}
          &bull;{" "}
          <span className={`${baseClass}__sha256`}>
            <TooltipWrapper
              tipContent={<>The software&apos;s SHA-256 hash.</>}
              position="top"
              showArrow
              underline={false}
            >
              {sha256.slice(0, 7)}&hellip;
            </TooltipWrapper>
            <div className={`${baseClass}__sha-copy-button`}>
              <Button variant="icon" iconStroke onClick={onCopySha256}>
                <Icon name="copy" />
              </Button>
            </div>
            <div className={`${baseClass}__copy-overlay`}>
              {copyMessage && (
                <div
                  className={`${baseClass}__copy-message`}
                >{`${copyMessage} `}</div>
              )}
            </div>
          </span>
        </>
      ) : (
        ""
      );
    };

    return (
      <>
        {renderInstallerDisplayText(installerType, isFma, androidPlayStoreId)}
        {renderVersionInfo()}
        {renderTimeStamp()}
        {renderSha256()}
      </>
    );
  };

  return (
    <div className={classNames}>
      {renderIcon()}
      <div className={`${baseClass}__info`}>
        <InstallerName name={softwareName} />
        <div className={`${baseClass}__details`}>{renderDetails()}</div>
      </div>
    </div>
  );
};

export default InstallerDetailsWidget;
