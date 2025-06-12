/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React from "react";
import classnames from "classnames";

import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "software-details-widget";

interface ISoftwareNameProps {
  name: string;
}

const SoftwareName = ({ name }: ISoftwareNameProps) => {
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

interface ISoftwareDetailsWidget {
  className?: string;
  softwareName: string;
  installerType: "package" | "vpp";
  addedTimestamp?: string;
  versionInfo?: JSX.Element;
}

const SoftwareDetailsWidget = ({
  className,
  softwareName,
  installerType,
  addedTimestamp,
  versionInfo,
}: ISoftwareDetailsWidget) => {
  const classNames = classnames(baseClass, className);

  const renderIcon = () => {
    return installerType === "package" ? (
      <Graphic name="file-pkg" />
    ) : (
      <SoftwareIcon name="appStore" size="medium" />
    );
  };

  const renderDetails = () => {
    return !addedTimestamp ? (
      versionInfo
    ) : (
      <>
        {versionInfo} &bull;{" "}
        <TooltipWrapper
          tipContent={internationalTimeFormat(new Date(addedTimestamp))}
          underline={false}
        >
          {addedFromNow(addedTimestamp)}
        </TooltipWrapper>
      </>
    );
  };

  return (
    <div className={classNames}>
      {renderIcon()}
      <div className={`${baseClass}__info`}>
        <SoftwareName name={softwareName} />
        <span className={`${baseClass}__details`}>{renderDetails()}</span>
      </div>
    </div>
  );
};

export default SoftwareDetailsWidget;
