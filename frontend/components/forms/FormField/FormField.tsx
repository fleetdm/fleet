import React from "react";
import classnames from "classnames";
import { isEmpty } from "lodash";

const baseClass = "form-field";

export interface IFormFieldProps {
  children: JSX.Element;
  className: string;
  error: string;
  hint: Array<any> | JSX.Element | string;
  label: Array<any> | JSX.Element | string;
  name: string;
  type: string;
}

// class FormField extends Component {
const FormField = ({
  children,
  className,
  error,
  hint,
  label,
  name,
  type,
}: IFormFieldProps) => {
  // static propTypes = {
  //   children: PropTypes.node,
  //   className: PropTypes.string,
  //   error: PropTypes.string,
  //   hint: PropTypes.oneOfType([
  //     PropTypes.array,
  //     PropTypes.node,
  //     PropTypes.string,
  //   ]),
  //   label: PropTypes.oneOfType([
  //     PropTypes.array,
  //     PropTypes.string,
  //     PropTypes.node,
  //   ]),
  //   name: PropTypes.string,
  //   type: PropTypes.string,
  // };

  const renderLabel = () => {
    // const { error, label, name } = this.props;

    const labelWrapperClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !isEmpty(error),
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

  const renderHint = () => {
    // const { hint } = this.props;

    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  // render() {
  // const { renderLabel, renderHint } = this;
  // const { children, className, type } = this.props;

  const formFieldClass = classnames(
    baseClass,
    {
      [`${baseClass}--${type}`]: !isEmpty(type),
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
  // }
};

export default FormField;
