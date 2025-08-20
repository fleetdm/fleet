import React from "react";
import { RouteComponentProps } from "react-router";

const baseClass = "script-batch-details";

interface IScriptBatchDetailsRouteParams {
  id: string;
}

type IScriptBatchDetailsProps = RouteComponentProps<
  undefined,
  IScriptBatchDetailsRouteParams
>;

const ScriptBatchDetails = ({
  router,
  routeParams,
  location,
}: IScriptBatchDetailsProps) => {
  return (
    <div className={`${baseClass}`}>
      <>TODO</>
    </div>
  );
};

export default ScriptBatchDetails;
