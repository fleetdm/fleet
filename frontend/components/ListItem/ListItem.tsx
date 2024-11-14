import React from "react";
import classnames from "classnames";

import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";

const baseClass = "list-item";

export type ISupportedGraphicNames = Extract<
  GraphicNames,
  | "file-configuration-profile"
  | "file-sh"
  | "file-ps1"
  | "file-py"
  | "file-script"
  | "file-pdf"
  | "file-pkg"
  | "file-p7m"
  | "file-pem"
>;

/**
 * A generic ListItem component that can be used to display a list of items. It
 * encapsulates the UI logic and styling for displaying a graphic, title,
 * details, and actions.
 */
interface IListItemProps {
  /** The grahpic you want to display for this list item. */
  graphic: ISupportedGraphicNames;
  title: string | JSX.Element;
  details: React.ReactNode;
  /** A collection of React Nodes that will render as list item actions. Can be
   * used to render buttons, links, etc.
   */
  actions: React.ReactNode;
  className?: string;
}

const ListItem = ({
  graphic,
  title,
  details,
  actions,
  className,
}: IListItemProps) => {
  const classNames = classnames(baseClass, className);
  return (
    <div className={classNames}>
      <div className={`${baseClass}__main-content`}>
        <Graphic name={graphic} />
        <div className={`${baseClass}__info`}>
          <span className={`${baseClass}__title`}>{title}</span>
          <div className={`${baseClass}__details`}>{details}</div>
        </div>
      </div>
      <div className={`${baseClass}__actions`}>{actions}</div>
    </div>
  );
};

export default ListItem;
