import React, { useState } from "react";
import classNames from "classnames";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children?: React.ReactNode;
  className?: string;
  /** default light purple */
  color?: "purple" | "purple-bold-border" | "yellow" | "grey";
  pageLevel?: boolean;
  /** cta and link are mutually exclusive */
  cta?: JSX.Element;
  /** closable and link are mutually exclusive */
  closable?: boolean;
  link?: string;
}

const InfoBanner = ({
  children,
  className,
  color = "purple",
  pageLevel,
  cta,
  closable,
  link,
}: IInfoBannerProps): JSX.Element => {
  const wrapperClasses = classNames(
    baseClass,
    `${baseClass}__${color}`,
    {
      [`${baseClass}__page-banner`]: !!pageLevel,
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
                name="ex"
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

  return <div className={wrapperClasses}>{content}</div>;
};

export default InfoBanner;
