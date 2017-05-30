import React, { Component, PropTypes } from 'react';
import Checkbox from 'components/forms/fields/Checkbox';
import decoratorInterface from 'interfaces/decorators';
import { includes } from 'lodash';
import DecoratorRow from 'components/decorators/DecoratorRows/DecoratorRow';

const baseClass = 'decorator-rows';

class DecoratorRows extends Component {
  static propTypes = {
    decorators: PropTypes.arrayOf(decoratorInterface),
    onCheckDecorator: PropTypes.func,
    onCheckAll: PropTypes.func,
    onSelectDecorator: PropTypes.func,
    allChecked: PropTypes.bool,
    onDoubleClick: PropTypes.func,
    checkedDecoratorIDs: PropTypes.arrayOf(PropTypes.number),
    selectedDecorator: decoratorInterface,
  };

  constructor (props) {
    super(props);
    this.state = { allDecoratorsChecked: false };
  }

  onCheck = (checked, id) => {
    const { allDecoratorsChecked } = this.state;
    const { onCheckDecorator } = this.props;
    if (allDecoratorsChecked) {
      this.setState({ allDecoratorsChecked: false });
    }
    onCheckDecorator(checked, id);
  }

  handleCheckAll = (checked) => {
    const { onCheckAll } = this.props;
    onCheckAll(checked);
  }

  isChecked = (decorator) => {
    const { checkedDecoratorIDs } = this.props;
    return includes(checkedDecoratorIDs, decorator.id);
  }

  render () {
    const {
      decorators,
      allChecked,
      onSelectDecorator,
      onDoubleClick,
      selectedDecorator,
     } = this.props;

    return (
      <div className={baseClass} >
        <table className={`${baseClass}__table`}>
          <thead>
            <tr>
              <th>
                <Checkbox
                  name="check-all-decorators"
                  onChange={this.handleCheckAll}
                  value={allChecked}
                />
              </th>
              <th>Decorator Name</th>
              <th>Type</th>
              <th>Interval</th>
              <th>Last Modified</th>
              <th>Query</th>
            </tr>
          </thead>
          <tbody>
            {decorators.map((decorator) => {
              return (
                <DecoratorRow
                  decorator={decorator}
                  key={`decorator-row-${decorator.id}`}
                  checked={this.isChecked(decorator)}
                  selected={selectedDecorator && selectedDecorator.id === decorator.id}
                  onCheck={this.onCheck}
                  onSelect={onSelectDecorator}
                  onDoubleClick={onDoubleClick}
                  builtIn={decorator.built_in}
                />
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default DecoratorRows;
