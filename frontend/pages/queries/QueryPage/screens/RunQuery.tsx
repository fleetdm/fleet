import React, { useState, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";
import SockJS from "sockjs-client";

// @ts-ignore
import { formatSelectedTargetsForApi } from "fleet/helpers"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions"; // @ts-ignore
import campaignHelpers from "redux/nodes/entities/campaigns/helpers";
import queryAPI from "services/entities/queries"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import { BASE_URL, DEFAULT_CAMPAIGN_STATE } from "utilities/constants"; // @ts-ignore
import local from "utilities/local"; // @ts-ignore
import { ICampaign, ICampaignState } from "interfaces/campaign";
import { IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";

import QueryResults from "../components/QueryResults";

interface IRunQueryProps {
  typedQueryBody: string;
  storedQuery: IQuery | undefined;
  selectedTargets: ITarget[];
}

const RunQuery = ({
  typedQueryBody,
  storedQuery,
  selectedTargets,
}: IRunQueryProps) => {
  const dispatch = useDispatch();

  const [isQueryFinished, setIsQueryFinished] = useState<boolean>(false);
  const [campaignState, setCampaignState] = useState<ICampaignState>(
    DEFAULT_CAMPAIGN_STATE
  );

  const ws = useRef(null);
  const runQueryInterval = useRef<any>(null);
  const globalSocket = useRef<any>(null);
  const previousSocketData = useRef<any>(null);

  const removeSocket = () => {
    if (globalSocket.current) {
      globalSocket.current.close();
      globalSocket.current = null;
      previousSocketData.current = null;
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
      setCampaignState((prevCampaignState) => ({
        ...prevCampaignState,
        campaign: returnedCampaign,
        queryIsRunning: true,
      }));

      websocket?.send(
        JSON.stringify({
          type: "auth",
          data: { token: local.getItem("auth_token") },
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

      if (
        socketData.type === "status" &&
        socketData.data.status === "finished"
      ) {
        return teardownDistributedQuery();
      }
    };
  };

  const onRunQuery = debounce(async () => {
    const sql = typedQueryBody || storedQuery?.query;

    if (!sql) {
      dispatch(
        renderFlash(
          "error",
          "Something went wrong running your query. Please try again."
        )
      );
      return false;
    }

    const selected = formatSelectedTargetsForApi(selectedTargets);
    removeSocket();
    destroyCampaign();

    try {
      const returnedCampaign = await queryAPI.run({ query: sql, selected });
      connectAndRunLiveQuery(returnedCampaign);
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
    }
  });

  useEffect(() => {
    onRunQuery();
  }, []);

  const onStopQuery = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    return teardownDistributedQuery();
  };

  const { campaign } = campaignState;
  return (
    <QueryResults
      campaign={campaign}
      onRunQuery={onRunQuery}
      onStopQuery={onStopQuery}
      isQueryFinished={isQueryFinished}
      dispatch={dispatch}
    />
  );
};

export default RunQuery;
