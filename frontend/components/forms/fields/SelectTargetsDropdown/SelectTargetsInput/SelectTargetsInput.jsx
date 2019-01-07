import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { difference } from 'lodash';
import Select from 'react-select';
import 'react-select/dist/react-select.css';

import targetInterface from 'interfaces/target';

class SelectTargetsInput extends Component {
  static propTypes = {
    className: PropTypes.string,
    disabled: PropTypes.bool,
    isLoading: PropTypes.bool,
    menuRenderer: PropTypes.func,
    onClose: PropTypes.func,
    onOpen: PropTypes.func,
    onFocus: PropTypes.func,
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
      className,
      disabled,
      isLoading,
      menuRenderer,
      onClose,
      onOpen,
      onFocus,
      onTargetSelect,
      onTargetSelectInputChange,
      selectedTargets,
      targets,
    } = this.props;

    return (
      <Select
        className={`${className} target-select`}
        disabled={disabled}
        isLoading={isLoading}
        filterOptions={this.filterOptions}
        labelKey="display_text"
        menuRenderer={menuRenderer}
        multi
        name="targets"
        options={targets}
        onChange={onTargetSelect}
        onClose={onClose}
        onOpen={onOpen}
        onFocus={onFocus}
        onInputChange={onTargetSelectInputChange}
        placeholder="Label Name, Host Name, IP Address, etc."
        resetValue={[]}
        scrollMenuIntoView={false}
        tabSelectsValue={false}
        value={selectedTargets}
        valueKey="display_text"
      />
    );
  }
}

export default SelectTargetsInput;
