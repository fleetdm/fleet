import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import moment from "moment";

import Checkbox from "components/forms/fields/Checkbox";
import ClickableTableRow from "components/ClickableTableRow";
import { isEqual } from "lodash";
import queryInterface from "interfaces/query";

const baseClass = "queries-list-row";

class QueriesListRow extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    onCheck: PropTypes.func.isRequired,
    onSelect: PropTypes.func.isRequired,
    onDoubleClick: PropTypes.func,
    query: queryInterface.isRequired,
    selected: PropTypes.bool,
    isOnlyObserver: PropTypes.bool,
  };

  shouldComponentUpdate(nextProps) {
    if (isEqual(nextProps, this.props)) {
      return false;
    }

    return true;
  }

  onCheck = (value) => {
    const { onCheck: handleCheck, query } = this.props;

    return handleCheck(value, query.id);
  };

  onSelect = () => {
    const { onSelect: handleSelect, query } = this.props;

    return handleSelect(query);
  };

  onDblClick = () => {
    const { onDoubleClick: handleDblClick, query } = this.props;

    return handleDblClick(query);
  };

  render() {
    const { checked, query, selected, isOnlyObserver } = this.props;
    const { onCheck, onSelect, onDblClick } = this;
    const {
      author_name: authorName,
      id,
      name,
      updated_at: updatedAt,
      description,
      observer_can_run,
    } = query;
    const lastModifiedDate = moment(updatedAt).format("MM/DD/YY");
    const rowClassName = classnames(baseClass, {
      [`${baseClass}--selected`]: selected,
    });

    return (
      <ClickableTableRow
        className={rowClassName}
        onClick={onSelect}
        onDoubleClick={onDblClick}
      >
        <td>
          <Checkbox
            name={`query-checkbox-${id}`}
            onChange={onCheck}
            value={checked}
          />
        </td>
        <td className={`${baseClass}__name`}>{name}</td>
        <td className={`${baseClass}__description`}>{description}</td>
        {isOnlyObserver ? null : (
          <td className={`${baseClass}__observers-can-run`}>
            {observer_can_run.toString()}
          </td>
        )}
        <td className={`${baseClass}__author-name`}>{authorName || "---"}</td>
        <td>{lastModifiedDate}</td>
      </ClickableTableRow>
    );
  }
}

export default QueriesListRow;
