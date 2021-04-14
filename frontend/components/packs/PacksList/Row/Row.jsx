import React, { Component } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";
import { isEqual } from "lodash";
import moment from "moment";

import Checkbox from "components/forms/fields/Checkbox";
import ClickableTableRow from "components/ClickableTableRow";
import KolideIcon from "components/icons/KolideIcon";
import packInterface from "interfaces/pack";

const baseClass = "packs-list-row";

class Row extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    onCheck: PropTypes.func,
    onDoubleClick: PropTypes.func,
    onSelect: PropTypes.func,
    pack: packInterface.isRequired,
    selected: PropTypes.bool,
  };

  shouldComponentUpdate(nextProps) {
    return !isEqual(this.props, nextProps);
  }

  handleChange = (shouldCheck) => {
    const { onCheck, pack } = this.props;

    return onCheck(shouldCheck, pack.id);
  };

  handleSelect = () => {
    const { onSelect, pack } = this.props;

    return onSelect(pack);
  };

  handleDoubleClick = () => {
    const { onDoubleClick, pack } = this.props;

    return onDoubleClick(pack);
  };

  renderStatusData = () => {
    const { disabled } = this.props.pack;
    const iconClassName = classNames(`${baseClass}__status-icon`, {
      [`${baseClass}__status-icon--enabled`]: !disabled,
      [`${baseClass}__status-icon--disabled`]: disabled,
    });

    if (disabled) {
      return (
        <td className={`${baseClass}__td`}>
          <KolideIcon className={iconClassName} name="offline" />
          <span className={`${baseClass}__status-text`}>Disabled</span>
        </td>
      );
    }

    return (
      <td className={`${baseClass}__td`}>
        <KolideIcon className={iconClassName} name="success-check" />
        <span className={`${baseClass}__status-text`}>Enabled</span>
      </td>
    );
  };

  render() {
    const { checked, pack, selected } = this.props;
    const {
      handleChange,
      handleDoubleClick,
      handleSelect,
      renderStatusData,
    } = this;
    const updatedTime = moment(pack.updated_at).format("MM/DD/YY");
    const rowClass = classNames(baseClass, {
      [`${baseClass}--selected`]: selected,
    });

    return (
      <ClickableTableRow
        className={rowClass}
        onClick={handleSelect}
        onDoubleClick={handleDoubleClick}
      >
        <td className={`${baseClass}__td`}>
          <Checkbox
            name={`select-pack-${pack.id}`}
            onChange={handleChange}
            value={checked}
            wrapperClassName={`${baseClass}__checkbox`}
          />
        </td>
        <td className={`${baseClass}__td ${baseClass}__td-pack-name`}>
          {pack.name}
        </td>
        <td className={`${baseClass}__td ${baseClass}__td-query-count`}>
          {pack.query_count}
        </td>
        {renderStatusData()}
        <td className={`${baseClass}__td ${baseClass}__td-host-count`}>
          {pack.total_hosts_count}
        </td>
        <td className={`${baseClass}__td`}>{updatedTime}</td>
      </ClickableTableRow>
    );
  }
}

export default Row;
