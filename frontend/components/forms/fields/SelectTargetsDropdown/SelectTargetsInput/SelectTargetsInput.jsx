import React, { Component } from "react";
import PropTypes from "prop-types";
import { difference, isEqual, uniqueId } from "lodash";
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

  constructor(props) {
    super(props);

    this.state = {
      uuidTargets: props.targets,
      uuidSelectedTargets: props.selectedTargets,
    };
  }

  // disable the eslint rule because it's the best way we can
  // fix #4905 without rewriting code that will be replaced
  // by the newer SelectTargets component soon.
  /* eslint-disable react/no-did-update-set-state */
  componentDidUpdate(prevProps) {
    const { targets, selectedTargets } = this.props;

    if (!isEqual(prevProps.targets, targets)) {
      // must have unique key to select correctly
      const uuidTargets = targets.map((target) => ({
        ...target,
        uuid: uniqueId(),
      }));

      this.setState({ uuidTargets });
    }

    if (!isEqual(prevProps.selectedTargets, selectedTargets)) {
      // must have unique key to deselect correctly
      const uuidSelectedTargets = selectedTargets.map((target) => ({
        ...target,
        uuid: uniqueId(),
      }));

      this.setState({ uuidSelectedTargets });
    }
  }

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
    } = this.props;
    const { uuidTargets, uuidSelectedTargets } = this.state;
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
        options={uuidTargets}
        onChange={onTargetSelect}
        onClose={onClose}
        onOpen={onOpen}
        onFocus={onFocus}
        onInputChange={handleInputChange}
        placeholder="Label name, host name, private IP address, etc."
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
