import React from "react";

import Button from "components/buttons/Button";
import { ILabel } from "interfaces/label";
import classnames from "classnames";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import { LABEL_DISPLAY_MAP } from "utilities/constants";

const baseClass = "labels-card";

interface ILabelsProps {
  onLabelClick: (label: ILabel) => void;
  labels: ILabel[];
  className?: string;
}

const Labels = ({
  onLabelClick,
  labels,
  className,
}: ILabelsProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  const labelItems = labels.map((label: ILabel) => {
    return (
      <li className="list__item" key={label.id}>
        <Button
          onClick={() => onLabelClick(label)}
          variant="pill"
          className="list__button"
        >
          {label.label_type === "builtin" && label.name in LABEL_DISPLAY_MAP
            ? LABEL_DISPLAY_MAP[label.name as keyof typeof LABEL_DISPLAY_MAP]
            : label.name}
        </Button>
      </li>
    );
  });

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
      className={classNames}
    >
      <CardHeader header="Labels" />
      {labels.length === 0 ? (
        <p className="info-flex__item">
          No labels are associated with this host.
        </p>
      ) : (
        <ul className="list">{labelItems}</ul>
      )}
    </Card>
  );
};

export default Labels;
