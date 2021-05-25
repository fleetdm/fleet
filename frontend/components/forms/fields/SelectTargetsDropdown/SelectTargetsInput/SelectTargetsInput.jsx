import React, { Component } from "react";
import PropTypes from "prop-types";
import { difference } from "lodash";
import Select from "react-select";
import "react-select/dist/react-select.css";

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
        onInputChange={handleInputChange}
        placeholder="Label name, host name, IP address, etc."
        resetValue={[]}
        scrollMenuIntoView={false}
        tabSelectsValue={false}
        value={selectedTargets}
        valueKey="id"
      />
    );
  }
}

export default SelectTargetsInput;
