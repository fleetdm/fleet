import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { isEqual, noop } from "lodash";

import Kolide from "kolide";
import targetInterface from "interfaces/target";
import { formatSelectedTargetsForApi } from "kolide/helpers";
import Input from "./SelectTargetsInput";
import Menu from "./SelectTargetsMenu";

const baseClass = "target-select";

class SelectTargetsDropdown extends Component {
  static propTypes = {
    disabled: PropTypes.bool,
    error: PropTypes.string,
    label: PropTypes.string,
    onFetchTargets: PropTypes.func,
    onSelect: PropTypes.func.isRequired,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    targetsCount: PropTypes.number,
  };

  static defaultProps = {
    disabled: false,
    onFetchTargets: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      isEmpty: false,
      isLoadingTargets: false,
      moreInfoTarget: null,
      query: "",
      targets: [],
    };
  }

  componentWillMount() {
    this.mounted = true;
    this.wrapperHeight = 0;
    this.fetchTargets();

    return false;
  }

  componentWillReceiveProps(nextProps) {
    const { selectedTargets } = nextProps;
    const { query } = this.state;

    if (!isEqual(selectedTargets, this.props.selectedTargets)) {
      this.fetchTargets(query, selectedTargets);
    }
  }

  componentWillUnmount() {
    this.mounted = false;
  }

  onInputClose = () => {
    const { document } = global;
    const coreWrapper = document.querySelector(".core-wrapper");

    this.setState({ moreInfoTarget: null, query: "" });
    coreWrapper.style.height = "auto";

    return false;
  };

  onInputFocus = () => {
    const { document } = global;
    this.wrapperHeight = document.querySelector(".core-wrapper").scrollHeight;

    return false;
  };

  onInputOpen = () => {
    const { document } = global;
    const { wrapperHeight } = this;

    const lookForOuterMenu = setInterval(() => {
      if (document.querySelectorAll(".Select-menu-outer")) {
        clearInterval(lookForOuterMenu);
        const coreWrapper = document.querySelector(".core-wrapper");

        const currentWrapperHeight = coreWrapper.scrollHeight;
        if (wrapperHeight < currentWrapperHeight) {
          coreWrapper.style.height = `${
            wrapperHeight + (currentWrapperHeight - wrapperHeight) + 15
          }px`;
        }
      }
    }, 5);

    return false;
  };

  onTargetSelectMoreInfo = (moreInfoTarget) => {
    return (evt) => {
      evt.preventDefault();

      const currentMoreInfoTarget = this.state.moreInfoTarget || {};

      if (isEqual(moreInfoTarget.id, currentMoreInfoTarget.id)) {
        return false;
      }

      this.setState({ moreInfoTarget });

      return false;
    };
  };

  onBackToResults = () => {
    this.setState({ moreInfoTarget: null });
  };

  fetchTargets = (query = "", selectedTargets = this.props.selectedTargets) => {
    const { onFetchTargets } = this.props;

    if (!this.mounted) {
      return false;
    }

    this.setState({ isLoadingTargets: true, query });

    return Kolide.targets
      .loadAll(query, formatSelectedTargetsForApi(selectedTargets))
      .then((response) => {
        const { targets } = response;
        const isEmpty = targets.length === 0;

        if (!this.mounted) {
          return false;
        }

        if (isEmpty) {
          // We don't want the lib's default "No Results" so we fake it
          targets.push({});
        }

        onFetchTargets(query, response);

        this.setState({
          isEmpty,
          isLoadingTargets: false,
          targets,
        });

        return query;
      })
      .catch((error) => {
        console.error("Error getting targets:", error);

        if (this.mounted) {
          this.setState({ isLoadingTargets: false });
        }
      });
  };

  renderLabel = () => {
    const { error, label, targetsCount } = this.props;

    const labelClassName = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: error,
    });

    if (!label) {
      return false;
    }

    return (
      <p className={labelClassName}>
        <span className={`${baseClass}__select-targets`}>{error || label}</span>
        <span className={`${baseClass}__targets-count`}>
          {" "}
          {targetsCount} unique {targetsCount === 1 ? "host" : "hosts"}
        </span>
      </p>
    );
  };

  render() {
    const { isEmpty, isLoadingTargets, moreInfoTarget, targets } = this.state;
    const {
      fetchTargets,
      onBackToResults,
      onInputClose,
      onInputOpen,
      onInputFocus,
      onTargetSelectMoreInfo,
      renderLabel,
    } = this;
    const { disabled, onSelect, selectedTargets } = this.props;
    const menuRenderer = Menu(
      onTargetSelectMoreInfo,
      moreInfoTarget,
      onBackToResults
    );

    const inputClasses = classnames({
      "show-preview": moreInfoTarget,
      "is-empty": isEmpty,
    });

    return (
      <div className={baseClass}>
        {renderLabel()}
        <Input
          className={inputClasses}
          disabled={disabled}
          isLoading={isLoadingTargets}
          menuRenderer={menuRenderer}
          onClose={onInputClose}
          onOpen={onInputOpen}
          onFocus={onInputFocus}
          onTargetSelect={onSelect}
          onTargetSelectInputChange={fetchTargets}
          selectedTargets={selectedTargets}
          targets={targets}
        />
      </div>
    );
  }
}

export default SelectTargetsDropdown;
