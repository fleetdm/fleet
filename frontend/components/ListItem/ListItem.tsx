import React from "react";

import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";

const baseClass = "list-item";

interface IListItemProps {
  graphic: GraphicNames;
  title: string;
  details: React.ReactNode;
  actions: React.ReactNode;
}

const ListItem = ({ graphic, title, details, actions }: IListItemProps) => {
  return (
    <div className={baseClass}>
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
