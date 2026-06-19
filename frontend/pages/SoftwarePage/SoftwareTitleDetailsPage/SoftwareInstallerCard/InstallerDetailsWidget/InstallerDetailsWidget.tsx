/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React, { useState } from "react";
import classnames from "classnames";

import { stringToClipboard } from "utilities/copy_text";
import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { InstallerType } from "interfaces/software";

import { isAndroidWebApp } from "pages/SoftwarePage/helpers";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import CustomLink from "components/CustomLink";
import AndroidLatestVersionWithTooltip from "components/MDM/AndroidLatestVersionWithTooltip";

const baseClass = "installer-details-widget";

interface IInstallerNameProps {
  name: string;
  /** When true, suppress the truncation tooltip — used in contexts (e.g. inactive
   * LibraryItemAccordion rows) where the row itself is non-interactive. */
  disableTooltip?: boolean;
}

const InstallerName = ({ name, disableTooltip }: IInstallerNameProps) => {
  const titleRef = React.useRef<HTMLDivElement>(null);
  const isTruncated = useCheckTruncatedElement(titleRef);

  return (
    <TooltipWrapper
      tipContent={name}
      position="top"
      underline={false}
      disableTooltip={disableTooltip || !isTruncated}
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
  androidPlayStoreId?: string
) => {
  if (installerType === "package") {
    return isFma ? "Fleet-maintained" : "Custom package";
  }
  if (androidPlayStoreId) {
    if (isAndroidWebApp(androidPlayStoreId)) {
      return "Web app";
    }

    return "Google Play Store";
  }
  return "App Store (VPP)";
};

interface IInstallerDetailsWidgetProps {
  className?: string;
  softwareName: string;
  installerType: InstallerType;
  addedTimestamp?: string;
  version?: string | null;
  sha256?: string | null;
  isFma: boolean;
  isLatestFmaVersion?: boolean;
  isScriptPackage: boolean;
  androidPlayStoreId?: string;
  customDetails?: string;
  /** Suppress the leading installer-type label ("Custom package", "App Store (VPP)",
   * etc.). Used when the widget is embedded somewhere that already conveys the type
   * (e.g. LibraryItemAccordion, where the icon + container do the same work). */
  hideInstallerType?: boolean;
  /** Suppress the title's truncation tooltip. Used by inactive LibraryItemAccordion
   * rows where the row itself is non-interactive. */
  disableTitleTooltip?: boolean;
}

const InstallerDetailsWidget = ({
  className,
  softwareName,
  installerType,
  addedTimestamp,
  sha256,
  version,
  isFma,
  isLatestFmaVersion = false,
  isScriptPackage,
  androidPlayStoreId,
  customDetails,
  hideInstallerType = false,
  disableTitleTooltip = false,
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
    if (customDetails) {
      return <>{customDetails}</>;
    }

    // Renders just the version chip (or null when hidden). The leading " · "
    // separator is added by the caller so that callers who suppress the
    // preceding type label don't get a stray middot.
    const renderVersionChip = (): React.ReactNode => {
      // Hide version info from script package and Android Play Store web apps
      if (isScriptPackage || isAndroidWebApp(androidPlayStoreId)) {
        return null;
      }

      if (androidPlayStoreId) {
        return (
          <AndroidLatestVersionWithTooltip
            androidPlayStoreId={androidPlayStoreId}
          />
        );
      }

      if (!version) {
        return (
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

      if (isFma) {
        return (
          <TooltipWrapper
            tipContent={
              <span>
                You can change the version in <strong>Actions &gt; Edit</strong>{" "}
                software.
              </span>
            }
          >
            <span>
              {version} {isLatestFmaVersion ? "(latest)" : ""}
            </span>
          </TooltipWrapper>
        );
      }

      if (installerType === "app-store") {
        return (
          <TooltipWrapper tipContent={<span>Updated every hour.</span>}>
            <span>{version}</span>
          </TooltipWrapper>
        );
      }

      return <span>{version}</span>;
    };

    const renderTimeStampChip = (): React.ReactNode =>
      addedTimestamp ? (
        <TooltipWrapper
          tipContent={internationalTimeFormat(new Date(addedTimestamp))}
          underline={false}
        >
          {addedFromNow(addedTimestamp)}
        </TooltipWrapper>
      ) : null;

    const renderSha256Chip = (): React.ReactNode => {
      return sha256 ? (
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
            <Button
              variant="icon"
              size="small"
              iconStroke
              onClick={onCopySha256}
            >
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
      ) : null;
    };

    const parts: React.ReactNode[] = [];
    if (!hideInstallerType) {
      parts.push(
        renderInstallerDisplayText(installerType, isFma, androidPlayStoreId)
      );
    }
    const versionChip = renderVersionChip();
    if (versionChip) parts.push(versionChip);
    const timeStampChip = renderTimeStampChip();
    if (timeStampChip) parts.push(timeStampChip);
    const sha256Chip = renderSha256Chip();
    if (sha256Chip) parts.push(sha256Chip);

    return parts.map((part, i) => (
      // eslint-disable-next-line react/no-array-index-key
      <React.Fragment key={i}>
        {i > 0 && <> &bull; </>}
        {part}
      </React.Fragment>
    ));
  };

  return (
    <div className={classNames}>
      {renderIcon()}
      <div className={`${baseClass}__info`}>
        <InstallerName
          name={softwareName}
          disableTooltip={disableTitleTooltip}
        />
        <div className={`${baseClass}__details`}>{renderDetails()}</div>
      </div>
    </div>
  );
};

export default InstallerDetailsWidget;
