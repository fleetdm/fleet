import React, { Component } from "react";
import PropTypes from "prop-types";
import { difference } from "lodash";
import Select from "react-select";
import "react-select/dist/react-select.css";
import { v4 as uuidv4 } from "uuid";

import debounce from "utilities/debounce";
import targetInterface from "interfaces/target";

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
  };

  handleInputChange = debounce(
    (query) => {
      const { onTargetSelectInputChange } = this.props;
      onTargetSelectInputChange(query);
    },
    { leading: false, trailing: true }
  );

  render() {
    const {
      className,
      disabled,
      isLoading,
      menuRenderer,
      onClose,
      onOpen,
      onFocus,
      onTargetSelect,
      selectedTargets,
      targets,
    } = this.props;

    const { handleInputChange } = this;

    // must have unique key to select correctly
    const uuidTargets = targets.map((target) => ({
      ...target,
      uuid: uuidv4(),
    }));

    // must have unique key to deselect correctly
    const uuidSelectedTargets = selectedTargets.map((target) => ({
      ...target,
      uuid: uuidv4(),
    }));

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
        options={uuidTargets}
        onChange={onTargetSelect}
        onClose={onClose}
        onOpen={onOpen}
        onFocus={onFocus}
        onInputChange={handleInputChange}
        placeholder="Label name, host name, IP address, etc."
        resetValue={[]}
        scrollMenuIntoView={false}
        tabSelectsValue={false}
        value={uuidSelectedTargets}
        valueKey="uuid" // must be unique, target ids are not unique
      />
    );
  }
}

export default SelectTargetsInput;
