import React from "react";
import classnames from "classnames";

const baseClass = "card";

type BorderRadiusSize = "small" | "medium" | "large" | "xlarge" | "xxlarge";
type CardColor = "white" | "gray" | "purple" | "yellow";

interface ICardProps {
  children?: React.ReactNode;
  /** The size of the border radius. Defaults to `small`.
   *
   * These correspond to the boarder radius in the design system. Look at
   * `var/_global.scss` for values */
  borderRadiusSize?: BorderRadiusSize;
  /** Includes the card shadows. Defaults to `false` */
  includeShadow?: boolean;
  /** The color of the card. Defaults to `white` */
  color?: CardColor;
  className?: string;
  /** The size of the padding around the content of the card. Defaults to `large`.
   *
   * These correspond to the padding sizes in the design system. Look at
   * `padding.scss` for values */
  paddingSize?: "small" | "medium" | "large" | "xlarge" | "xxlarge";
}

/**
 * A generic card component that will be used to render content within a card with a border and
 * and selected background color.
 */
const Card = ({
  children,
  borderRadiusSize = "small",
  includeShadow = false,
  color = "white",
  className,
  paddingSize = "large",
}: ICardProps) => {
  const classNames = classnames(
    baseClass,
    `${baseClass}__${color}`,
    `${baseClass}__radius-${borderRadiusSize}`,
    {
      [`${baseClass}__padding-${paddingSize}`]: paddingSize !== undefined,
      [`${baseClass}__shadow`]: includeShadow,
    },
    className
  );

  return <div className={classNames}>{children}</div>;
};

export default Card;
