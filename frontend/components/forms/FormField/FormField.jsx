import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

const baseClass = "form-field";

class FormField extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
    error: PropTypes.string,
    hint: PropTypes.oneOfType([
      PropTypes.array,
      PropTypes.node,
      PropTypes.string,
    ]),
    label: PropTypes.oneOfType([
      PropTypes.array,
      PropTypes.string,
      PropTypes.node,
    ]),
    name: PropTypes.string,
    type: PropTypes.string,
  };

  renderLabel = () => {
    const { error, label, name } = this.props;
    const labelWrapperClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: error,
    });

    if (!label) {
      return false;
    }

    return (
      <label className={labelWrapperClasses} htmlFor={name}>
        {error || label}
      </label>
    );
  };

  renderHint = () => {
    const { hint } = this.props;

    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  render() {
    const { renderLabel, renderHint } = this;
    const { children, className, type } = this.props;

    const formFieldClass = classnames(
      baseClass,
      {
        [`${baseClass}--${type}`]: type,
      },
      className
    );

    return (
      <div className={formFieldClass}>
        {renderLabel()}
        {children}
        {renderHint()}
      </div>
    );
  }
}

export default FormField;
