import React, { Component, PropTypes } from 'react';
import { difference } from 'lodash';
import Select from 'react-select';
import 'react-select/dist/react-select.css';

import targetInterface from '../../../interfaces/target';

class SelectTargetsInput extends Component {
  static propTypes = {
    isLoading: PropTypes.bool,
    menuRenderer: PropTypes.func,
    onTargetSelect: PropTypes.func,
    onTargetSelectInputChange: PropTypes.func,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targets: PropTypes.arrayOf(targetInterface),
  };

  filterOptions = (options) => {
    const { selectedTargets } = this.props;

    return difference(options, selectedTargets);
  }

  render () {
    const {
      isLoading,
      menuRenderer,
      onTargetSelect,
      onTargetSelectInputChange,
      selectedTargets,
      targets,
    } = this.props;

    return (
      <Select
        className="target-select"
        isLoading={isLoading}
        filterOptions={this.filterOptions}
        labelKey="display_text"
        menuRenderer={menuRenderer}
        multi
        name="targets"
        options={targets}
        onChange={onTargetSelect}
        onInputChange={onTargetSelectInputChange}
        placeholder="Label Name, Host Name, IP Address, etc."
        resetValue={[]}
        value={selectedTargets}
        valueKey="display_text"
      />
    );
  }
}

export default SelectTargetsInput;
