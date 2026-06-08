import React from "react";
import classnames from "classnames";

import { GraphicNames, GRAPHIC_MAP } from "components/graphics";

interface IGraphicProps {
  name: GraphicNames;
  /** scale-40-24 Workaround to scale 40px to 24px */
  className?: string;
}

const baseClass = "graphic";

const Graphic = ({ name, className }: IGraphicProps) => {
  const classNames = classnames(baseClass, className);

  const GraphicComponent = GRAPHIC_MAP[name];

  return (
    <div className={classNames} data-testid={`${name}-graphic`}>
      <GraphicComponent />
    </div>
  );
};

export default Graphic;
