import classnames from "classnames";
import React from "react";

import Card from "components/Card";
import CardHeader from "components/CardHeader";
import Tag from "components/Tag";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { ILabel } from "interfaces/label";

const baseClass = "host-labels-card";

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

  const labelItems = labels
    .filter((label: ILabel) => label.label_type !== "builtin")
    .map((label: ILabel) => {
      return (
        <li className={`${baseClass}__list-item`} key={label.id}>
          <Tag
            type="clickable"
            onClick={() => onLabelClick(label)}
            className={`${baseClass}__list-button`}
          >
            <TooltipTruncatedText value={label.name} />
          </Tag>
        </li>
      );
    });

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      className={classNames}
    >
      <CardHeader header="Labels" />
      {labelItems.length === 0 ? (
        <p className="info-flex__item">
          No labels are associated with this host.
        </p>
      ) : (
        <ul className={`${baseClass}__list`}>{labelItems}</ul>
      )}
    </Card>
  );
};

export default Labels;
