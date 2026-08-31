/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React from "react";
import classnames from "classnames";

import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import { InstallerType, SoftwareSource } from "interfaces/software";

import { isAndroidWebApp } from "pages/SoftwarePage/helpers";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import AndroidLatestVersionWithTooltip from "components/MDM/AndroidLatestVersionWithTooltip";

const baseClass = "installer-details-widget";

interface IInstallerNameProps {
  name: string;
  /** When true, suppress the truncation tooltip — used in contexts (e.g.
   * inactive LibraryItemAccordion rows) where every tooltip on the row is
   * suppressed. */
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
  isFma: boolean;
  isLatestFmaVersion?: boolean;
  isScriptPackage: boolean;
  /** Software source, used to pick the file icon (e.g. `file-py` for `py_packages`). */
  source?: SoftwareSource;
  androidPlayStoreId?: string;
  customDetails?: string;
  /** Suppress the leading installer-type label ("Custom package", "App Store (VPP)",
   * etc.). Used when the widget is embedded somewhere that already conveys the type
   * (e.g. LibraryItemAccordion, where the icon + container do the same work). */
  hideInstallerType?: boolean;
  /** Suppress every hover tooltip the widget would normally render (title
   * truncation, FMA "change in Actions > Edit" hint, App Store "Updated every
   * hour", Android Play Store link, "Fleet couldn't read the version", and the
   * `addedAt` formatted-time tooltip). Used by inactive LibraryItemAccordion
   * rows, whose outer wrapper already shows the rollback hover tooltip — Fleet
   * UI avoids stacking two tooltips on the same hover target across the app,
   * so the widget's tooltips have to defer to the row-level one. */
  disableTooltips?: boolean;
}

const InstallerDetailsWidget = ({
  className,
  softwareName,
  installerType,
  addedTimestamp,
  version,
  isFma,
  isLatestFmaVersion = false,
  isScriptPackage,
  source,
  androidPlayStoreId,
  customDetails,
  hideInstallerType = false,
  disableTooltips = false,
}: IInstallerDetailsWidgetProps) => {
  const classNames = classnames(baseClass, className);

  const renderIcon = () => {
    if (installerType === "app-store") {
      if (androidPlayStoreId) {
        return <SoftwareIcon name="androidPlayStore" size="medium" />;
      }
      return <SoftwareIcon name="appleAppStore" size="medium" />;
    }
    if (source === "py_packages") {
      return <Graphic name="file-py" />;
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
        // AndroidLatestVersionWithTooltip has no disable-tooltip prop, so for
        // inactive rows we render the plain "Latest" text instead — keeps the
        // chip readable without bringing in the Play Store hover tooltip.
        if (disableTooltips) return <span>Latest</span>;
        return (
          <AndroidLatestVersionWithTooltip
            androidPlayStoreId={androidPlayStoreId}
          />
        );
      }

      if (!version) {
        return (
          <TooltipWrapper
            disableTooltip={disableTooltips}
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
            disableTooltip={disableTooltips}
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
          <TooltipWrapper
            disableTooltip={disableTooltips}
            tipContent={<span>Updated every hour.</span>}
          >
            <span>{version}</span>
          </TooltipWrapper>
        );
      }

      return <span>{version}</span>;
    };

    const renderTimeStampChip = (): React.ReactNode =>
      addedTimestamp ? (
        <TooltipWrapper
          disableTooltip={disableTooltips}
          tipContent={internationalTimeFormat(new Date(addedTimestamp))}
          underline={false}
        >
          {addedFromNow(addedTimestamp)}
        </TooltipWrapper>
      ) : null;

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
        <InstallerName name={softwareName} disableTooltip={disableTooltips} />
        <div className={`${baseClass}__details`}>{renderDetails()}</div>
      </div>
    </div>
  );
};

export default InstallerDetailsWidget;
