import React, { useState } from "react";
import classNames from "classnames";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children?: React.ReactNode;
  className?: string;
  /** default light purple */
  color?: "purple" | "purple-bold-border" | "yellow" | "grey";
  /** default 4px  */
  borderRadius?: "large" | "xlarge";
  pageLevel?: boolean;
  /** Add this element to the end of the banner message. Mutually exclusive with `link`. */
  cta?: JSX.Element;
  /** closable and link are mutually exclusive */
  closable?: boolean;
  /** Makes the entire banner clickable */
  link?: string;
  icon?: IconNames;
}

const InfoBanner = ({
  children,
  className,
  color = "purple",
  borderRadius,
  pageLevel,
  cta,
  closable,
  link,
  icon,
}: IInfoBannerProps) => {
  const wrapperClasses = classNames(
    baseClass,
    `${baseClass}__${color}`,
    {
      [`${baseClass}__${color}`]: !!color,
      [`${baseClass}__border-radius-${borderRadius}`]: !!borderRadius,
      [`${baseClass}__page-banner`]: !!pageLevel,
      [`${baseClass}__icon`]: !!icon,
    },
    className
  );

  const [hideBanner, setHideBanner] = useState(false);

  const content = (
    <>
      <div className={`${baseClass}__info`}>{children}</div>

      {(cta || closable) && (
        <div className={`${baseClass}__cta`}>
          {cta}
          {closable && (
            <Button variant="unstyled" onClick={() => setHideBanner(true)}>
              <Icon
                name="close"
                color="core-fleet-black"
                size="small"
                className={`${baseClass}__close`}
              />
            </Button>
          )}
        </div>
      )}
    </>
  );

  if (hideBanner) {
    return <></>;
  }

  if (link) {
    return (
      <a
        href={link}
        target="_blank"
        rel="noreferrer"
        className={wrapperClasses}
      >
        {content}
      </a>
    );
  }

  return (
    <div className={wrapperClasses} role="status">
      {content}
    </div>
  );
};

export default InfoBanner;
