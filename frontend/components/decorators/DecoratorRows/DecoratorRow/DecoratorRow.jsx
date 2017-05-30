import React, { Component, PropTypes } from 'react';
import ClickableTableRow from 'components/ClickableTableRow';
import Checkbox from 'components/forms/fields/Checkbox';
import decoratorInterface from 'interfaces/decorators';
import classnames from 'classnames';
import moment from 'moment';
import { isEqual } from 'lodash';


const baseClass = 'decorator-row';

class DecoratorRow extends Component {
  static propTypes = {
    checked: PropTypes.bool,
    onCheck: PropTypes.func,
    onSelect: PropTypes.func,
    onDoubleClick: PropTypes.func,
    decorator: decoratorInterface,
    selected: PropTypes.bool,
    builtIn: PropTypes.bool,
  };

  shouldComponentUpdate (nextProps) {
    if (isEqual(nextProps, this.props)) {
      return false;
    }
    return true;
  }

  onCheck = (value) => {
    const { onCheck: handleCheck, decorator } = this.props;
    return handleCheck(value, decorator.id);
  }

  onSelect = () => {
    const { onSelect: handleSelect, decorator } = this.props;
    // built in can't be selected
    if (decorator.built_in) {
      return false;
    }
    return handleSelect(decorator);
  }

  onDoubleClick = () => {
    const { onDoubleClick: handleDoubleClick, decorator } = this.props;
    if (decorator.built_in) {
      return false;
    }
    return handleDoubleClick(decorator);
  }

  render () {
    const { onCheck, onSelect, onDoubleClick } = this;
    const { selected, checked, decorator, builtIn } = this.props;
    const { id, name, updated_at: updatedAt, query, type, interval } = decorator;
    const lastModifiedDate = moment(updatedAt).format('MM/DD/YY');
    const rowClassName = classnames(baseClass, {
      [`${baseClass}--selected`]: selected,
    });
    return (
      <ClickableTableRow className={rowClassName} onClick={onSelect} onDoubleClick={onDoubleClick} >
        <td>
          <Checkbox
            name={`decorator-checkbox-${id}`}
            onChange={onCheck}
            value={checked}
            disabled={builtIn}
          />
        </td>
        <td className={`${baseClass}__name`}>{name}</td>
        <td className={`${baseClass}__name`}>{type}</td>
        <td className={`${baseClass}__name`}>{interval}</td>
        <td>{lastModifiedDate}</td>
        <td className={`${baseClass}__name`}>{query}</td>
      </ClickableTableRow>
    );
  }

}

export default DecoratorRow;
