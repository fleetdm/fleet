import React, { useState, useEffect, useRef, useContext } from "react";
import SockJS from "sockjs-client";

import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";
import { formatSelectedTargetsForApi } from "utilities/helpers";

import queryAPI from "services/entities/queries";
import campaignHelpers from "utilities/campaign_helpers";
import debounce from "utilities/debounce";
import { BASE_URL, DEFAULT_CAMPAIGN_STATE } from "utilities/constants";

import { authToken } from "utilities/local";

import { ICampaign, ICampaignState } from "interfaces/campaign";
import { IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";

import QueryResults from "../../edit/components/QueryResults";

const RESPONSE_COUNT_ZERO = { results: 0, errors: 0 } as const;
const CAMPAIGN_LIMIT = 250000;

interface IRunQueryProps {
  storedQuery: IQuery | undefined;
  selectedTargets: ITarget[];
  queryId: number | null;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const RunQuery = ({
  storedQuery,
  selectedTargets,
  queryId,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IRunQueryProps): JSX.Element | null => {
  const { lastEditedQueryBody } = useContext(QueryContext);
  const { renderFlash } = useContext(NotificationContext);

  const [isQueryFinished, setIsQueryFinished] = useState(false);
  const [isQueryClipped, setIsQueryClipped] = useState(false);
  const [campaignState, setCampaignState] = useState<ICampaignState>(
    DEFAULT_CAMPAIGN_STATE
  );

  const isStoredQueryEdited = storedQuery?.query !== lastEditedQueryBody;

  const ws = useRef(null);
  const runQueryInterval = useRef<any>(null);
  const globalSocket = useRef<any>(null);
  const previousSocketData = useRef<any>(null);
  const responseCount = useRef({ ...RESPONSE_COUNT_ZERO });

  const removeSocket = () => {
    if (globalSocket.current) {
      globalSocket.current.close();
      globalSocket.current = null;
      previousSocketData.current = null;
      responseCount.current = RESPONSE_COUNT_ZERO;
    }
  };

  const setupDistributedQuery = (socket: WebSocket | null) => {
    globalSocket.current = socket;
    const update = () => {
      setCampaignState((prevCampaignState) => ({
        ...prevCampaignState,
        runQueryMilliseconds: prevCampaignState.runQueryMilliseconds + 1000,
      }));
    };

    if (!runQueryInterval.current) {
      runQueryInterval.current = setInterval(update, 1000);
    }
  };

  const teardownDistributedQuery = () => {
    if (runQueryInterval.current) {
      clearInterval(runQueryInterval.current);
      runQueryInterval.current = null;
    }

    setCampaignState((prevCampaignState) => ({
      ...prevCampaignState,
      queryIsRunning: false,
      runQueryMilliseconds: 0,
    }));
    setIsQueryFinished(true);
    removeSocket();
  };

  const destroyCampaign = () => {
    setCampaignState(DEFAULT_CAMPAIGN_STATE);
  };

  const connectAndRunLiveQuery = (returnedCampaign: ICampaign) => {
    let { current: websocket }: { current: WebSocket | null } = ws;
    websocket = new SockJS(`${BASE_URL}/v1/fleet/results`, undefined, {});
    websocket.onopen = () => {
      setupDistributedQuery(websocket);
      // `prevCampaignState` at this point is the default state. Update that with what we get from
      // the API response
      setCampaignState((prevCampaignState) => ({
        ...prevCampaignState,
        campaign: { ...prevCampaignState.campaign, returnedCampaign },
        queryIsRunning: true,
      }));

      websocket?.send(
        JSON.stringify({
          type: "auth",
          data: { token: authToken() },
        })
      );
      websocket?.send(
        JSON.stringify({
          type: "select_campaign",
          data: { campaign_id: returnedCampaign.id },
        })
      );
    };

    websocket.onmessage = ({ data }: { data: string }) => {
      // string is easy to compare before converting to object
      if (data === previousSocketData.current) {
        return false;
      }

      previousSocketData.current = data;
      const socketData = JSON.parse(data);
      setCampaignState((prevCampaignState) => {
        return {
          ...prevCampaignState,
          ...campaignHelpers.updateCampaignState(socketData)(prevCampaignState),
        };
      });
      responseCount.current.results += socketData?.data?.rows?.length ?? 0;
      responseCount.current.errors += socketData?.data?.error ? 1 : 0;

      if (
        socketData.type === "status" &&
        socketData.data.status === "finished"
      ) {
        return teardownDistributedQuery();
      }
      if (
        responseCount.current.results + responseCount.current.errors >=
        CAMPAIGN_LIMIT
      ) {
        teardownDistributedQuery();
        setIsQueryClipped(true);
      }
    };
  };

  const onRunQuery = debounce(async () => {
    if (!lastEditedQueryBody) {
      renderFlash(
        "error",
        "Something went wrong running your query. Please try again."
      );
      return false;
    }

    const selected = formatSelectedTargetsForApi(selectedTargets);
    setIsQueryFinished(false);
    removeSocket();
    destroyCampaign();

    try {
      const returnedCampaign = await queryAPI.run({
        query: lastEditedQueryBody,
        queryId: isStoredQueryEdited ? null : queryId, // we treat edited SQL as a new query
        selected,
      });

      connectAndRunLiveQuery(returnedCampaign);
    } catch (campaignError: any) {
      const err = campaignError.toString();
      if (err.includes("no hosts targeted")) {
        renderFlash(
          "error",
          "Your target selections did not include any hosts. Please try again."
        );
      } else if (err.includes("resource already created")) {
        renderFlash(
          "error",
          "A campaign with the provided query text has already been created"
        );
      } else if (err.includes("forbidden") || err.includes("unauthorized")) {
        renderFlash(
          "error",
          "It seems you do not have the rights to run this query. If you believe this is in error, please contact your administrator."
        );
      } else {
        renderFlash("error", "Something has gone wrong. Please try again.");
      }

      return teardownDistributedQuery();
    }
  });

  const onStopQuery = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    return teardownDistributedQuery();
  };

  useEffect(() => {
    onRunQuery();
  }, []);

  const { campaign } = campaignState;
  return (
    <QueryResults
      campaign={campaign}
      onRunQuery={onRunQuery}
      onStopQuery={onStopQuery}
      isQueryFinished={isQueryFinished}
      isQueryClipped={isQueryClipped}
      setSelectedTargets={setSelectedTargets}
      goToQueryEditor={goToQueryEditor}
      queryName={storedQuery?.name}
      targetsTotalCount={targetsTotalCount}
    />
  );
};

export default RunQuery;
