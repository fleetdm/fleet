import React from "react";

import Button from "components/buttons/Button";
import { ILabel } from "interfaces/label";
import { enforceFleetSentenceCasing } from "utilities/strings/stringUtils";
import classnames from "classnames";

const baseClass = "labels-card";

interface ILabelsProps {
  onLabelClick: (label: ILabel) => void;
  labels: ILabel[];
}

const Labels = ({ onLabelClick, labels }: ILabelsProps): JSX.Element => {
  const classNames = classnames(baseClass, "section", "labels");

  const labelItems = labels.map((label: ILabel) => {
    return (
      <li className="list__item" key={label.id}>
        <Button
          onClick={() => onLabelClick(label)}
          variant="label"
          className="list__button"
        >
          {enforceFleetSentenceCasing(label.name)}
        </Button>
      </li>
    );
  });

  return (
    <div className={classNames}>
      <p className="section__header">Labels</p>
      {labels.length === 0 ? (
        <p className="info-flex__item">
          No labels are associated with this host.
        </p>
      ) : (
        <ul className="list">{labelItems}</ul>
      )}
    </div>
  );
};

export default Labels;
