import React from "react";

import Button from "components/buttons/Button";
import { ILabel } from "interfaces/label";

interface ILabelsProps {
  onLabelClick: (label: ILabel) => void;
  labels: ILabel[];
}

const Labels = ({ onLabelClick, labels }: ILabelsProps): JSX.Element => {
  const labelItems = labels.map((label: ILabel) => {
    return (
      <li className="list__item" key={label.id}>
        <Button
          onClick={() => onLabelClick(label)}
          variant="label"
          className="list__button"
        >
          {label.name}
        </Button>
      </li>
    );
  });

  return (
    <div className="section labels col-50">
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
