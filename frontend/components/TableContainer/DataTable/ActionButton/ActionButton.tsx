import React from "react";
import { useCallback } from "react";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Button from "../../../buttons/Button";

const baseClass = "action-button";

interface IActionButtonProps {
  callback: (evt: any) => void | undefined;
  name: string;
  selectedRows: any[];
}

function useActionCallback(
  callbackFn: (selectedRows: any[]) => void | undefined
) {
  return useCallback(
    (selectedRows) => {
      const entityIds = selectedRows.map((row: any) => row.original.id);
      console.log("callback called: ", entityIds);
      callbackFn(entityIds);
    },
    [callbackFn]
  );
}

const ActionButton = (props: IActionButtonProps): JSX.Element => {
  const { callback, name, selectedRows } = props;
  const onActionClick = useActionCallback(callback);

  return (
    <Button onClick={() => onActionClick(selectedRows)} variant={"text-link"}>
      {name}
    </Button>
  );
};

export default ActionButton;
