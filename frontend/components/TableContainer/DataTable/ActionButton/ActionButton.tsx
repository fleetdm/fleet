import React from "react";
import PropTypes from "prop-types";
import { useCallback } from "react";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Button from "../../../buttons/Button";

// const baseClass = "action-button";

export interface IActionButtonProps {
  callback: (targetIds: number[]) => void | undefined;
  name: string;
  hideButton?: boolean | ((val: any) => boolean);
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
  const testCondition = (
    prop: boolean | ((val: any) => boolean) | undefined
  ) => {
    console.log("test condition called");
    if (typeof prop === "function") {
      return prop;
    }
    return () => Boolean(prop);
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
