import React, { Component, PropTypes } from 'react';
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
        menuRenderer={menuRenderer}
        multi
        name="targets"
        options={targets}
        onChange={onTargetSelect}
        onInputChange={onTargetSelectInputChange}
        placeholder="Label Name, Host Name, IP Address, etc."
        resetValue={[]}
        value={selectedTargets}
        valueKey="label"
      />
    );
  }
}

export default SelectTargetsInput;
