import React, { useState } from "react";
import { Dispatch } from "redux";
import moment from "moment";
import FileSaver from "file-saver";
import { filter, isEqual } from "lodash";

// @ts-ignore
import Fleet from "fleet";
import { formatSelectedTargetsForApi } from "fleet/helpers"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions"; // @ts-ignore
import campaignHelpers from "redux/nodes/entities/campaigns/helpers";
import queryAPI from "services/entities/queries"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import convertToCSV from "utilities/convert_to_csv";
import { ICampaign } from "interfaces/campaign";
import { IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";

// @ts-ignore
import QueryProgressDetails from "components/queries/QueryProgressDetails"; // @ts-ignore
import QueryResultsTable from "components/queries/QueryResultsTable";

interface IRunQueryProps {
  baseClass: string;
  typedQueryBody: string;
  storedQuery: IQuery | undefined;
  campaign: ICampaign | null;
  selectedTargets: ITarget[];
  queryIsRunning: boolean;
  setQueryIsRunning: (value: boolean) => void;
  setCampaign: (value: ICampaign | null) => void;
  dispatch: Dispatch;
};

let runQueryInterval: any = null;
let globalSocket: any = null;
let previousSocketData: any = null;

const QUERY_RESULTS_OPTIONS = {
  FULL_SCREEN: "FULL_SCREEN",
  SHRINKING: "SHRINKING",
};

const RunQuery = ({
  baseClass,
  typedQueryBody,
  storedQuery,
  campaign,
  selectedTargets,
  queryIsRunning,
  setQueryIsRunning,
  setCampaign,
  dispatch,
}: IRunQueryProps) => {
  const [isReady, setIsReady] = useState<boolean>(false);
  const [runQueryMilliseconds, setRunQueryMilliseconds] = useState<number>(0);
  const [csvQueryName, setCsvQueryName] = useState<string>("Query Results");
  
  const removeSocket = () => {
    if (globalSocket) {
      globalSocket.close();
      globalSocket = null;
      previousSocketData = null;
    }

    return false;
  };
  
  const setupDistributedQuery = (socket: any) => {
    globalSocket = socket;
    const update = () => {
      setRunQueryMilliseconds(runQueryMilliseconds + 1000);
    };

    if (!runQueryInterval) {
      runQueryInterval = setInterval(update, 1000);
    }

    return false;
  };

  const teardownDistributedQuery = () => {
    if (runQueryInterval) {
      clearInterval(runQueryInterval);
      runQueryInterval = null;
    }

    setQueryIsRunning(false);
    setRunQueryMilliseconds(0);
    removeSocket();

    return false;
  };

  const destroyCampaign = () => {
    setCampaign(null);

    return false;
  };
  
  const onRunQuery = debounce(async () => {
    const sql = typedQueryBody || storedQuery?.query;

    if (!sql) {
      dispatch(renderFlash("error", "Something went wrong running your query. Please try again."));
      return false;
    }

    const selected = formatSelectedTargetsForApi(selectedTargets);

    removeSocket();
    destroyCampaign();

    try {
      const campaignResponse = await queryAPI.run({ query: sql, selected });

      Fleet.websockets.queries.run(campaignResponse.id).then((socket: any) => {
        setupDistributedQuery(socket);
        setCampaign(campaignResponse);
        setQueryIsRunning(true);

        socket.onmessage = ({ data }: any) => {
          const socketData = JSON.parse(data);

          if (previousSocketData && isEqual(socketData, previousSocketData)) {
            return false;
          }

          previousSocketData = socketData;

          const {
            campaign: socketCampaign,
            queryIsRunning: socketQueryIsRunning,
          } = campaignHelpers.updateCampaignState(socketData);
          socketCampaign && setCampaign(socketCampaign);
          socketQueryIsRunning !== undefined &&
            setQueryIsRunning(socketQueryIsRunning);

          if (
            socketData.type === "status" &&
            socketData.data.status === "finished"
          ) {
            return teardownDistributedQuery();
          }

          return false;
        };
      });
    } catch (campaignError) {
      if (campaignError === "resource already created") {
        dispatch(
          renderFlash(
            "error",
            "A campaign with the provided query text has already been created"
          )
        );

        return false;
      }

      dispatch(renderFlash("error", campaignError));
      return false;
    }
  });

  const onStopQuery = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    return teardownDistributedQuery();
  };

  const onExportQueryResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (!campaign) {
      return false;
    }

    const { query_results: queryResults } = campaign;

    if (queryResults) {
      const csv = convertToCSV(queryResults, (fields: string[]) => {
        const result = filter(fields, (f) => f !== "host_hostname");
        result.unshift("host_hostname");

        return result;
      });

      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${csvQueryName} (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }

    return false;
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (!campaign) {
      return false;
    }

    const { errors } = campaign;

    if (errors) {
      const csv = convertToCSV(errors, (fields: string[]) => {
        const result = filter(fields, (f) => f !== "host_hostname");
        result.unshift("host_hostname");

        return result;
      });

      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${csvQueryName} Errors (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }

    return false;
  };

  if (isReady) {
    return (
      <QueryResultsTable
        campaign={campaign}
        onExportQueryResults={onExportQueryResults}
        onExportErrorsResults={onExportErrorsResults}
        // isQueryFullScreen={isQueryFullScreen}
        // isQueryShrinking={isQueryShrinking}
        // onToggleQueryFullScreen={onToggleQueryFullScreen}
        onRunQuery={onRunQuery}
        onStopQuery={onStopQuery}
        // onTargetSelect={onTargetSelect}
        queryIsRunning={queryIsRunning}
        queryTimerMilliseconds={runQueryMilliseconds}
      />
    );
  }

  return (
    <QueryProgressDetails
      campaign={campaign}
      onRunQuery={onRunQuery}
      onStopQuery={onStopQuery}
      queryIsRunning={queryIsRunning}
      queryTimerMilliseconds={runQueryMilliseconds}
      disableRun={false}
    />
  );
};

export default RunQuery;