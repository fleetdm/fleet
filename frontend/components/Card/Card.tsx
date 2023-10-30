import React from "react";
import classnames from "classnames";

const baseClass = "card";

type BorderRadiusSize = "small" | "medium" | "large";
type CardColor = "white" | "gray" | "purple" | "yellow";

interface ICardProps {
  children?: React.ReactNode;
  /** The size of the border radius. Defaults to `small` */
  borderRadiusSize?: BorderRadiusSize;
  /** Includes the card shadows. Defaults to `false` */
  includeShadow?: boolean;
  /** The color of the card. Defaults to `white` */
  color?: CardColor;
  className?: string;
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
}: ICardProps) => {
  const classNames = classnames(
    baseClass,
    `${baseClass}__${color}`,
    `${baseClass}__radius-${borderRadiusSize}`,
    { [`${baseClass}__shadow`]: includeShadow },
    className
  );

  return <div className={classNames}>{children}</div>;
};

export default Card;
