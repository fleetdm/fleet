import React, { useState, useEffect } from "react";
import { Link } from "react-router";
import classnames from "classnames";
import { connect, useDispatch } from "react-redux";
import { useQuery, useMutation } from "react-query";
import { push } from "react-router-redux";
import moment from "moment";
import FileSaver from "file-saver";

// @ts-ignore
import Fleet from "fleet";
import { formatSelectedTargetsForApi } from "fleet/helpers";
import queryAPI from "services/entities/queries";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import convertToCSV from "utilities/convert_to_csv"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import permissionUtils from "utilities/permissions";
import { IQueryFormData, IQuery } from "interfaces/query";
import { ITarget, ITargetsResponse } from "interfaces/target";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import { 
  selectOsqueryTable, 
  setSelectedTargets,
  setSelectedTargetsQuery // @ts-ignore
} from "redux/nodes/components/QueryPages/actions"; // @ts-ignore
import campaignHelpers from "redux/nodes/entities/campaigns/helpers"; // @ts-ignore
import QueryForm from "components/forms/queries/QueryForm1"; // @ts-ignore
import WarningBanner from "components/WarningBanner"; // @ts-ignore
import QueryPageSelectTargets from "components/queries/QueryPageSelectTargets"; // @ts-ignore
import QueryResultsTable from "components/queries/QueryResultsTable"; // @ts-ignore
import QuerySidePanel from "components/side_panels/QuerySidePanel"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import { hasSavePermissions, selectHosts } from "pages/queries/QueryPage1/helpers";

import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";
import { filter, isEqual } from "lodash";
import { IOsqueryTable } from "interfaces/osquery_table";
import { IUser } from "interfaces/user";
import { ICampaign } from "interfaces/campaign";

interface IQueryPageProps {
  queryIdForEdit: string;
  selectedTargets: ITarget[];
  selectedOsqueryTable: IOsqueryTable;
  currentUser: IUser;
  isBasicTier: boolean;
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

const QUERY_RESULTS_OPTIONS = {
  FULL_SCREEN: "FULL_SCREEN",
  SHRINKING: "SHRINKING",
};

const baseClass = "query-page";

const QueryPage = ({ 
  queryIdForEdit, 
  selectedTargets,
  selectedOsqueryTable,
  currentUser,
  isBasicTier,
}: IQueryPageProps) => {
  const { EDITOR, TARGETS, RUN, RESULTS } = PAGE_STEP;
  const dispatch = useDispatch();
  
  const [step, setStep] = useState<string>(EDITOR);
  const [typedQueryBody, setTypedQueryBody] = useState<string>('');
  const [runQueryMilliseconds, setRunQueryMilliseconds] = useState<number>(0);
  const [campaign, setCampaign] = useState<ICampaign | null>(null);
  const [queryIsRunning, setQueryIsRunning] = useState<boolean>(false);
  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [targetsError, setTargetsError] = useState<string | null>(null);
  const [queryResultsToggle, setQueryResultsToggle] = useState<any>(null);
  const [queryPosition, setQueryPosition] = useState<any>({});
  const [selectRelatedHostTarget, setSelectRelatedHostTarget] = useState<boolean>(true);
  const [observerShowSql, setObserverShowSql] = useState<boolean>(false);
  const [liveQueryError, setLiveQueryError] = useState<string>("");
  const [csvQueryName, setCsvQueryName] = useState<string>("Query Results");
  
  const { status, data: storedQuery, error }: { status: string, data: IQuery | undefined, error: any } = useQuery("query", () => queryAPI.load(queryIdForEdit), {
    enabled: !!queryIdForEdit
  });
  const { mutateAsync: createQuery } = useMutation((formData: IQueryFormData) => queryAPI.create(formData));

  useEffect(() => {
    const checkLiveQuery = () => {
      Fleet.status.live_query().catch((response: any) => {
        try {
          const error = response.message.errors[0].reason;
          setLiveQueryError(error);
        } catch (e) {
          const error = `Unknown error: ${e}`;
          setLiveQueryError(error);
        }
      });
    };

    checkLiveQuery();
  }, []);

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
    setCampaign(null);

    return false;
  };

  const onSaveQueryFormSubmit = debounce(async (formData: IQueryFormData) => {
    // const { error } = validateQuery(formData.query);

    // if (error) {
    //   dispatch(renderFlash("error", error));

    //   return false;
    // }

    try {
      const { query }: { query: IQuery } = await createQuery(formData);
      dispatch(push(PATHS.EDIT_QUERY(query)));
      dispatch(renderFlash("success", "Query created!"));
    } catch (error) {
      console.log(error);
      dispatch(renderFlash("error", "Something went wrong creating your query. Please try again."));
    }

    // const mutation = useMutation(() => queryAPI.create(formData), {
    //   onSuccess: (data) => {
    //     dispatch(push(PATHS.EDIT_QUERY(data)));
    //     dispatch(renderFlash("success", "Query created!"));
    //   },
    // });

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
    const sql = typedQueryBody || storedQuery?.query;
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

  const onStopQuery = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    return teardownDistributedQuery();
  };

  const onUpdateQuery = async (formData: IQueryFormData) => {
    if (!storedQuery) {
      return false;
    }
    
    const updatedQuery = deepDifference(formData, storedQuery);

    try {
      await queryAPI.update(storedQuery, updatedQuery);
      dispatch(renderFlash("success", "Query updated!"));
    } catch(error) {
      console.log(error);
      dispatch(renderFlash("error", "Something went wrong updating your query. Please try again."));
    }

    return false;
  };

  const onFetchTargets = (targetSearchText: string, targetResponse: ITargetsResponse) => {
    const { targets_count: targetsCount } = targetResponse;

    dispatch(setSelectedTargetsQuery(targetSearchText));
    setTargetsCount(targetsCount);

    return false;
  };

  const onTargetSelect = (selected: ITarget[]) => {
    setTargetsError(null);
    dispatch(setSelectedTargets(selectedTargets));

    return false;
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

  const renderLiveQueryWarning = () => {
    if (!liveQueryError) {
      return false;
    }

    return (
      <WarningBanner className={`${baseClass}__warning`} shouldShowWarning>
        <h2 className={`${baseClass}__warning-title`}>
          Live query request failed
        </h2>
        <p>
          <span>Error:</span> {liveQueryError}
        </p>
      </WarningBanner>
    );
  };

  const renderTargetsInput = () => {
    return (
      <QueryPageSelectTargets
        campaign={campaign}
        error={targetsError}
        onFetchTargets={onFetchTargets}
        onRunQuery={onRunQuery}
        onStopQuery={onStopQuery}
        onTargetSelect={onTargetSelect}
        queryIsRunning={queryIsRunning}
        selectedTargets={selectedTargets}
        targetsCount={targetsCount}
        queryTimerMilliseconds={runQueryMilliseconds}
        disableRun={liveQueryError !== undefined}
        queryId={queryIdForEdit}
        isBasicTier={isBasicTier}
      />
    );
  };

  const renderResultsTable = () => {
    // const loading = queryIsRunning && !campaign.hosts_count.total;
    const isQueryFullScreen =
      queryResultsToggle === QUERY_RESULTS_OPTIONS.FULL_SCREEN;
    const isQueryShrinking =
      queryResultsToggle === QUERY_RESULTS_OPTIONS.SHRINKING;
    const resultsClasses = classnames(`${baseClass}__results`, "body-wrap", {
      [`${baseClass}__results--loading`]: queryIsRunning,
      [`${baseClass}__results--full-screen`]: isQueryFullScreen,
    });

    // if (isEqual(campaign, DEFAULT_CAMPAIGN)) {
    //   return false;
    // }

    return (
      <div className={resultsClasses}>
        <QueryResultsTable
          campaign={campaign}
          onExportQueryResults={onExportQueryResults}
          onExportErrorsResults={onExportErrorsResults}
          isQueryFullScreen={isQueryFullScreen}
          isQueryShrinking={isQueryShrinking}
          // onToggleQueryFullScreen={onToggleQueryFullScreen}
          onRunQuery={onRunQuery}
          onStopQuery={onStopQuery}
          onTargetSelect={onTargetSelect}
          queryIsRunning={queryIsRunning}
          queryTimerMilliseconds={runQueryMilliseconds}
        />
      </div>
    );
  };

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
          <QueryForm
            // formData={storedQuery}
            onCreateQuery={onSaveQueryFormSubmit}
            onChangeFunc={onChangeQueryFormField}
            onOsqueryTableSelect={onOsqueryTableSelect}
            onRunQuery={onRunQuery}
            onStopQuery={onStopQuery}
            onUpdate={onUpdateQuery}
            queryIsRunning={queryIsRunning}
            serverErrors={error || {}}
            selectedOsqueryTable={selectedOsqueryTable}
            title={storedQuery?.name || "New query"}
            hasSavePermissions={hasSavePermissions(currentUser)}
          />
        </div>
        {renderLiveQueryWarning()}
        {/* ONLY SHOW FOR STEP 2 */}
        {/* {renderTargetsInput()} */}

        {/* ONLY SHOW FOR STEP 3 */}
        {/* {renderResultsTable()} */}
      </div>
      <QuerySidePanel
        onOsqueryTableSelect={onOsqueryTableSelect}
        selectedOsqueryTable={selectedOsqueryTable}
      />
    </div>
  );
};

const mapStateToProps = (state: any, { params }: any) => {
  const { id: queryIdForEdit } = params;
  const { selectedOsqueryTable, selectedTargets } = state.components.QueryPages;
  const currentUser = state.auth.user;
  const config = state.app.config;
  const isBasicTier = permissionUtils.isBasicTier(config);

  return { 
    queryIdForEdit,
    selectedTargets,
    selectedOsqueryTable,
    currentUser,
    isBasicTier,
  };
};

export default connect(mapStateToProps)(QueryPage);