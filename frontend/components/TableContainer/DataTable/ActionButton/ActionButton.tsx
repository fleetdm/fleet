import React, { useCallback } from "react";
import PropTypes from "prop-types";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Button from "../../../buttons/Button";

export interface IActionButtonProps {
  callback: (targetIds: number[]) => void | undefined;
  name: string;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  targetIds: number[];
  variant?: string;
}

function useActionCallback(
  callbackFn: (targetIds: number[]) => void | undefined
) {
  return useCallback(
    (targetIds) => {
      console.log("callback called: ", targetIds);
      callbackFn(targetIds);
    },
    [callbackFn]
  );
}

const ActionButton = (props: IActionButtonProps): JSX.Element | null => {
  const { callback, name, targetIds, variant, hideButton } = props;
  const onActionClick = useActionCallback(callback);

  // hideButton is intended to provide a flexible way to specify show/hide conditions via a boolean or a function that evaluates to a boolean
  // currently it is typed to accept an array of targetIds but this typing could easily be expanded to include other use cases
  const testCondition = (
    hideButtonProp: boolean | ((targetIds: number[]) => boolean) | undefined
  ) => {
    if (typeof hideButtonProp === "function") {
      return hideButtonProp;
    }
    return () => Boolean(hideButtonProp);
  };
  const isHidden = testCondition(hideButton)(targetIds);

  return !isHidden ? (
    <Button onClick={() => onActionClick(targetIds)} variant={variant}>
      {name}
    </Button>
  ) : null;
};

ActionButton.propTypes = {
  callback: PropTypes.func,
  name: PropTypes.string,
  hideButton: PropTypes.oneOfType([PropTypes.bool, PropTypes.func]),
  targetIds: PropTypes.arrayOf(PropTypes.number),
  variant: PropTypes.string,
};

export default ActionButton;
