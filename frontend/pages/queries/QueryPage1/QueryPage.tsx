import React, { useState } from "react";
import { Link } from "react-router";
import { connect, useDispatch } from "react-redux";
import { useQuery, useMutation } from "react-query";
import { push } from "react-router-redux";

// @ts-ignore
import Fleet from "fleet";
import { formatSelectedTargetsForApi } from "fleet/helpers";
import queryAPI from "services/entities/queries";
import PATHS from "router/paths";// @ts-ignore
import debounce from "utilities/debounce";
import { INewQuery, IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions"; // @ts-ignore
import { selectOsqueryTable } from "redux/nodes/components/QueryPages/actions"; // @ts-ignore
import campaignHelpers from "redux/nodes/entities/campaigns/helpers"; // @ts-ignore
import QueryForm from "components/forms/queries/QueryForm"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";
import { isEqual } from "lodash";

interface IQueryPageProps {
  queryId: string;
  selectedTargets: ITarget[];
};

interface ICampaign {
  hosts_count: {
    total: number;
  };
};

let runQueryInterval: any = null;
let globalSocket: any = null;
let previousSocketData: any = null;

const PAGE_STEP = {
  EDITOR: "EDITOR",
  TARGETS: "TARGETS",
  RUN: "RUN",
  RESULTS: "RESULTS",
};

const DEFAULT_CAMPAIGN: ICampaign = {
  hosts_count: {
    total: 0,
  }
};

const baseClass = "query-page";

const QueryPage = ({ queryId, selectedTargets }: IQueryPageProps) => {
  const { EDITOR, TARGETS, RUN, RESULTS } = PAGE_STEP;
  const dispatch = useDispatch();
  
  const [step, setStep] = useState<string>(EDITOR);
  const [typedQueryBody, setTypedQueryBody] = useState<string>('');
  const [runQueryMilliseconds, setRunQueryMilliseconds] = useState<number>(0);
  const [campaign, setCampaign] = useState<ICampaign>(DEFAULT_CAMPAIGN);
  const [queryIsRunning, setQueryIsRunning] = useState<boolean>(false);
  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [targetsError, setTargetsError] = useState<string | null>(null);
  const [queryResultsToggle, setQueryResultsToggle] = useState<any>(null);
  const [queryPosition, setQueryPosition] = useState<any>({});
  const [selectRelatedHostTarget, setSelectRelatedHostTarget] = useState<boolean>(true);
  const [observerShowSql, setObserverShowSql] = useState<boolean>(false);
  
  const { status, data: query, error }: { status: string, data: IQuery | undefined, error: any } = useQuery("query", () => queryAPI.load(queryId), {
    enabled: !!queryId
  });

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

    setQueryIsRunning(false),
    setRunQueryMilliseconds(0),
    removeSocket();

    return false;
  };

  const destroyCampaign = () => {
    setCampaign(DEFAULT_CAMPAIGN);

    return false;
  };

  const onSaveQueryFormSubmit = debounce((formData: INewQuery) => {
    const { error } = validateQuery(formData.query);

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    const mutation = useMutation(() => queryAPI.create(formData), {
      onSuccess: () => {
        query && dispatch(push(PATHS.EDIT_QUERY(query)));
        dispatch(renderFlash("success", "Query created!"));
      },
    });

    // return dispatch(queryActions.create(formData))
    //   .then((query) => {
    //     dispatch(push(PATHS.EDIT_QUERY(query)));
    //     dispatch(renderFlash("success", "Query created!"));
    //   })
    //   .catch(() => false);
  });

  const onChangeQueryFormField = (fieldName: string, value: string) => {
    if (fieldName === "query") {
      setTypedQueryBody(value);
    }

    return false;
  };

  const onOsqueryTableSelect = (tableName: string) => {
    dispatch(selectOsqueryTable(tableName));

    return false;
  };

  const onRunQuery = debounce(async () => {
    // const { queryText, targetsCount } = this.state;
    // const { query } = this.props.query;
    const sql = typedQueryBody || query?.query;
    // const { dispatch, selectedTargets } = this.props;
    const { error } = validateQuery(sql);

    if (!sql) {
      return false;
    }

    if (!selectedTargets.length) {
      setTargetsError("You must select at least one target to run a query");

      return false;
    }

    if (!targetsCount) {
      setTargetsError("You must select a target with at least one host to run a query");

      return false;
    }

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    const selected = formatSelectedTargetsForApi(selectedTargets);

    removeSocket();
    destroyCampaign();

    try {
      const campaignResponse = await queryAPI.run({ query: sql, selected });

      Fleet.websockets.queries
        .run(campaignResponse.id)
        .then((socket: any) => {
          setupDistributedQuery(socket);
          setCampaign(campaignResponse);
          setQueryIsRunning(true);

          socket.onmessage = ({ data }: any) => {
            const socketData = JSON.parse(data);

            if (
              previousSocketData &&
              isEqual(socketData, previousSocketData)
            ) {
              return false;
            }
            
            previousSocketData = socketData;
            
            const { campaign, queryIsRunning } = campaignHelpers.updateCampaignState(socketData)
            campaign && setCampaign(campaign);
            queryIsRunning !== undefined && setQueryIsRunning(queryIsRunning);

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
    };
  });

  return (
    <div className={`${baseClass} has-sidebar`}>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__form body-wrap`}>
          <Link
            to={PATHS.MANAGE_QUERIES}
            className={`${baseClass}__back-link`}
          >
            <img src={BackChevron} alt="back chevron" id="back-chevron" />
            <span>Back to queries</span>
          </Link>
          {/* <QueryForm
            formData={query}
            handleSubmit={onSaveQueryFormSubmit}
            onChangeFunc={onChangeQueryFormField}
            onOsqueryTableSelect={onOsqueryTableSelect}
            onRunQuery={onRunQuery}
            onStopQuery={onStopQuery}
            onUpdate={onUpdateQuery}
            queryIsRunning={queryIsRunning}
            serverErrors={errors}
            selectedOsqueryTable={selectedOsqueryTable}
            title={title}
            hasSavePermissions={hasSavePermissions(currentUser)}
          /> */}
        </div>
        {/* {renderLiveQueryWarning()}
        {renderTargetsInput()}
        {renderResultsTable()} */}
      </div>
      {/* <QuerySidePanel
        onOsqueryTableSelect={onOsqueryTableSelect}
        onTextEditorInputChange={onTextEditorInputChange}
        selectedOsqueryTable={selectedOsqueryTable}
      /> */}
    </div>
  );
};

const mapStateToProps = (_: any, { params }: any) => {
  const { id: queryId } = params;
  return { queryId };
};

export default connect(mapStateToProps)(QueryPage);