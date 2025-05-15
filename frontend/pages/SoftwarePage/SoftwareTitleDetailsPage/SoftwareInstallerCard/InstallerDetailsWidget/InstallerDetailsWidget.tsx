/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React, { useState } from "react";
import classnames from "classnames";
import { stringToClipboard } from "utilities/copy_text";

import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "installer-details-widget";

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

const renderInstallerDisplayText = (installerType: string, isFma: boolean) => {
  if (installerType === "package") {
    return isFma ? "Fleet-maintained" : "Custom package";
  }
  return "App Store (VPP)";
};

interface IInstallerDetailsWidgetProps {
  className?: string;
  softwareName: string;
  installerType: "package" | "vpp";
  addedTimestamp?: string;
  versionInfo?: JSX.Element;
  sha256?: string | null;
  isFma: boolean;
}

const InstallerDetailsWidget = ({
  className,
  softwareName,
  installerType,
  addedTimestamp,
  sha256,
  versionInfo,
  isFma,
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
    return installerType === "package" ? (
      <Graphic name="file-pkg" />
    ) : (
      <SoftwareIcon name="appStore" size="medium" />
    );
  };

  const renderDetails = () => {
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
        {renderInstallerDisplayText(installerType, isFma)} &bull; {versionInfo}
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
