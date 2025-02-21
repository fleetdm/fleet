import React from "react";
import classnames from "classnames";

interface ITabTextProps {
  className?: string;
  children: React.ReactNode;
  count?: number;
  /** Changes count badge from default purple to red */
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

  const countClassNames = classnames(`${baseClass}__count`, {
    [`${baseClass}__count--error`]: isErrorCount,
  });

  const renderCount = () => {
    if (count && count > 0) {
      return <div className={countClassNames}>{count.toLocaleString()}</div>;
    }
    return undefined;
  };

  return (
    <div className={classNames}>
      <div className={`${baseClass}__text}`} data-text={children}>
        {children}
      </div>
      {renderCount()}
    </div>
  );
};

export default TabText;
