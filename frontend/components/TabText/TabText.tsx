import React from "react";
import classnames from "classnames";
import { isError } from "lodash";

interface ITabTextProps {
  className?: string;
  children: React.ReactNode;
  count?: number;
  isErrorCount?: boolean;
}

/*
 * This component exists so we can unify the styles
 * and add styles to react-tab text.
 */
const baseClass = "tab-text";

const TabText = ({
  className,
  children,
  count,
  isErrorCount = false,
}: ITabTextProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  const countClassNames = classnames(`${baseClass}-count`, {
    [`${baseClass}-count--error`]: isErrorCount,
  });

  const renderCount = () => {
    if (count && count > 0) {
      return <div className={countClassNames}>{count.toLocaleString()}</div>;
    }
    return undefined;
  };

  return (
    <>
      <div className={classNames}>{children}</div>
      {renderCount()}
    </>
  );
};

export default TabText;
