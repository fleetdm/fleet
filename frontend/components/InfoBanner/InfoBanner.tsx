import React, { useState } from "react";
import classNames from "classnames";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";
import Card from "components/Card";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children?: React.ReactNode;
  className?: string;
  /** default light purple */
  color?: "purple" | "yellow" | "grey";
  /** default 4px  */
  borderRadius?: "medium" | "xlarge";
  pageLevel?: boolean;
  /** Add this element to the end of the banner message. Mutually exclusive with `link`. */
  cta?: JSX.Element;
  /** closable and link are mutually exclusive */
  closable?: boolean;
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
  icon,
}: IInfoBannerProps) => {
  const wrapperClasses = classNames(
    baseClass,
    {
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

  return (
    <Card
      className={wrapperClasses}
      color={color}
      borderRadiusSize={borderRadius}
    >
      {content}
    </Card>
  );
};

export default InfoBanner;
