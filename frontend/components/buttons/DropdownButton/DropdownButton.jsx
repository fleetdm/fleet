import React, { Component } from "react";
import PropTypes from "prop-types";
import { noop } from "lodash";
import classnames from "classnames";

import ClickOutside from "components/ClickOutside";
import Button from "components/buttons/Button";

const baseClass = "dropdown-button";

export class DropdownButton extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
    disabled: PropTypes.bool,
    options: PropTypes.arrayOf(
      PropTypes.shape({
        disabled: PropTypes.bool,
        label: PropTypes.string,
        onClick: PropTypes.func,
      })
    ).isRequired,
    size: PropTypes.string,
    tabIndex: PropTypes.number,
    type: PropTypes.string,
    variant: PropTypes.string,
  };

  static defaultProps = {
    onChange: noop,
  };

  constructor(props) {
    super(props);

    this.state = { isOpen: false };
  }

  setDOMNode = (DOMNode) => {
    this.DOMNode = DOMNode;
  };

  toggleDropdown = () => {
    const { isOpen } = this.state;
    this.setState({ isOpen: !isOpen });
  };

  optionClick = (evt, onClick) => {
    this.setState({ isOpen: false });
    onClick(evt);
  };

  renderOptions = (opt, idx) => {
    const { optionClick } = this;
    const { disabled, label, onClick } = opt;

    return (
      <li
        className={`${baseClass}__option`}
        key={`dropdown-button-option-${idx}`}
      >
        <Button
          variant="unstyled"
          onClick={(evt) => optionClick(evt, onClick)}
          disabled={disabled}
        >
          {label}
        </Button>
      </li>
    );
  };

  render() {
    const {
      children,
      className,
      disabled,
      options,
      size,
      tabIndex,
      type,
      variant,
    } = this.props;
    const { isOpen } = this.state;
    const { toggleDropdown, renderOptions, setDOMNode } = this;

    const buttonClass = classnames(baseClass, className);
    const optionsClass = classnames(`${baseClass}__options`, {
      [`${baseClass}__options--opened`]: isOpen,
    });

    return (
      <div className={`${baseClass}__wrapper`} ref={setDOMNode}>
        <Button
          className={`${buttonClass} downcarat`}
          disabled={disabled}
          onClick={toggleDropdown}
          size={size}
          tabIndex={tabIndex}
          type={type}
          variant={variant}
        >
          {children}{" "}
        </Button>

        <ul className={optionsClass}>
          {options.map((option, i) => {
            return renderOptions(option, i);
          })}
        </ul>
      </div>
    );
  }
}

export default ClickOutside(DropdownButton, {
  getDOMNode: (component) => {
    return component.DOMNode;
  },
  onOutsideClick: (component) => {
    return () => {
      component.setState({ isOpen: false });

      return false;
    };
  },
});
