import React from "react";
import classnames from "classnames";

const baseClass = "card";

type CardColors = "white" | "gray" | "purple" | "yellow";

interface ICardProps {
  children?: React.ReactNode;
  /** defaults to white */
  color?: CardColors;
  className?: string;
}

/**
 * A generic card component that will be used to render content within a card with a border and
 * and selected background color.
 */
const Card = ({ children, color = "white", className }: ICardProps) => {
  const classNames = classnames(baseClass, `${baseClass}__${color}`, className);

  return <div className={classNames}>{children}</div>;
};

export default Card;
