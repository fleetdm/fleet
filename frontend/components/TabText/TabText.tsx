import React from "react";
import classnames from "classnames";

import Icon from "components/Icon";

type TabCountVariant = "alert" | "pending";
interface ITabTextProps {
  className?: string;
  children: React.ReactNode;
  count?: number;
  countVariant?: TabCountVariant;
  /** When true, renders a green check icon next to the tab text
   * (e.g. to indicate that something is configured for this tab). */
  showCheck?: boolean;
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
  countVariant,
  showCheck,
}: ITabTextProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  const countClassNames = classnames(`${baseClass}__count`, {
    [`${baseClass}__count__alert`]: countVariant === "alert",
    [`${baseClass}__count__pending`]: countVariant === "pending",
  });

  const renderCount = () => {
    if (count && count > 0) {
      return <div className={countClassNames}>{count.toLocaleString()}</div>;
    }
    return undefined;
  };

  return (
    <div className={classNames}>
      <div className={`${baseClass}__text`} data-text={children}>
        {children}
      </div>
      {renderCount()}
      {showCheck && (
        <Icon
          name="check"
          size="small"
          color="core-fleet-green"
          className={`${baseClass}__check`}
        />
      )}
    </div>
  );
};

export default TabText;
