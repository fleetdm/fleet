import React, { useState } from "react";
import classNames from "classnames";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";
import Card from "components/Card";
import { Colors } from "styles/var/colors";

const baseClass = "info-banner";

export interface IInfoBannerProps {
  children?: React.ReactNode;
  className?: string;
  /** default grey */
  color?: "grey" | "yellow";
  /** default 4px  */
  borderRadius?: "medium" | "xlarge";
  pageLevel?: boolean;
  /** Add this element to the end of the banner message. Mutually exclusive with `link`. */
  cta?: JSX.Element;
  /** closable and link are mutually exclusive */
  closable?: boolean;
  /** Renders an icon to the left of the banner copy. When set, the banner
   * switches from `space-between` to a left-aligned flex layout so the icon
   * groups with the text rather than getting pushed to the opposite edge. */
  icon?: IconNames;
  /** Overrides the icon's default color when `icon` is set. */
  iconColor?: Colors;
}

const InfoBanner = ({
  children,
  className,
  color = "grey",
  borderRadius,
  pageLevel,
  cta,
  closable,
  icon,
  iconColor,
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
      {icon && (
        <Icon
          name={icon}
          color={iconColor}
          className={`${baseClass}__leading-icon`}
        />
      )}
      <div className={`${baseClass}__info`}>{children}</div>

      {(cta || closable) && (
        <div className={`${baseClass}__cta`}>
          {cta}
          {closable && (
            <Button variant="subdued" onClick={() => setHideBanner(true)}>
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
