/** TODO: This component is similar to other UI elements that can
 * be abstracted to use a shared base component (e.g. DetailsWidget) */

import React, { useLayoutEffect, useState } from "react";
import classnames from "classnames";

import { internationalTimeFormat } from "utilities/helpers";
import { addedFromNow } from "utilities/date_format";

import Graphic from "components/Graphic";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "software-details-widget";

/** TODO: pull this hook and SoftwareName component out. We could use this other places */
function useTruncatedElement<T extends HTMLElement>(ref: React.RefObject<T>) {
  const [isTruncated, setIsTruncated] = useState(false);

  useLayoutEffect(() => {
    const element = ref.current;
    function updateIsTruncated() {
      if (element) {
        const { scrollWidth, clientWidth } = element;
        setIsTruncated(scrollWidth > clientWidth);
      }
    }
    window.addEventListener("resize", updateIsTruncated);
    updateIsTruncated();
    return () => window.removeEventListener("resize", updateIsTruncated);
  }, [ref]);

  return isTruncated;
}

interface ISoftwareNameProps {
  name: string;
}

const SoftwareName = ({ name }: ISoftwareNameProps) => {
  const titleRef = React.useRef<HTMLDivElement>(null);
  const isTruncated = useTruncatedElement(titleRef);

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
