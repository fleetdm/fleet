import React, { useState, useEffect, useRef, useContext } from "react";
import SockJS from "sockjs-client";

import { PolicyContext } from "context/policy";
import { NotificationContext } from "context/notification";
import { formatSelectedTargetsForApi } from "utilities/helpers";

import campaignHelpers from "utilities/campaign_helpers";
import queryAPI from "services/entities/queries";
import debounce from "utilities/debounce";
import { BASE_URL, DEFAULT_CAMPAIGN_STATE } from "utilities/constants";
import { authToken } from "utilities/local";

import { ICampaign, ICampaignState } from "interfaces/campaign";
import { IPolicy } from "interfaces/policy";
import { ITarget } from "interfaces/target";

import PolicyResults from "../components/PolicyResults";

interface IRunQueryProps {
  storedPolicy: IPolicy | undefined;
  selectedTargets: ITarget[];
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const RunQuery = ({
  storedPolicy,
  selectedTargets,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IRunQueryProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [isQueryFinished, setIsQueryFinished] = useState(false);
  const [campaignState, setCampaignState] = useState<ICampaignState>(
    DEFAULT_CAMPAIGN_STATE
  );
  const { lastEditedQueryBody } = useContext(PolicyContext);

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

      if (
        socketData.type === "status" &&
        socketData.data.status === "finished"
      ) {
        return teardownDistributedQuery();
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
      // we do not want to run a stored query,
      // instead always run provided query
      const queryId = null;
      const returnedCampaign = await queryAPI.run({
        query: lastEditedQueryBody,
        queryId,
        selected,
      });

      connectAndRunLiveQuery(returnedCampaign);
    } catch (campaignError: any) {
      if (campaignError === "resource already created") {
        renderFlash(
          "error",
          "A campaign with the provided query text has already been created"
        );
      }

      if ("message" in campaignError) {
        const { message } = campaignError;

        if (message === "forbidden") {
          renderFlash(
            "error",
            "It seems you do not have the rights to run this query. If you believe this is in error, please contact your administrator."
          );
        } else {
          renderFlash("error", "Something has gone wrong. Please try again.");
        }
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
    <PolicyResults
      campaign={campaign}
      isQueryFinished={isQueryFinished}
      onRunQuery={onRunQuery}
      onStopQuery={onStopQuery}
      setSelectedTargets={setSelectedTargets}
      goToQueryEditor={goToQueryEditor}
      policyName={storedPolicy?.name}
      targetsTotalCount={targetsTotalCount}
    />
  );
};

export default RunQuery;
