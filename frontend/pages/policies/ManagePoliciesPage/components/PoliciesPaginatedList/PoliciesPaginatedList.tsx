import React, {
  useCallback,
  useImperativeHandle,
  useRef,
  useState,
  useContext,
  forwardRef,
  Ref,
} from "react";
import { ReactElement } from "react-markdown/lib/react-markdown";
import { AppContext } from "context/app";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import { useQuery } from "react-query";
import {
  IPolicy,
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPoliciesCountResponse,
} from "interfaces/policy";
import teamPoliciesAPI, {
  IPoliciesApiParams,
  IPoliciesCountApiParams,
} from "services/entities/team_policies";
import globalPoliciesAPI from "services/entities/global_policies";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

// Extend the IPolicy interface with some virtual properties that make it easier
// to track item state. These are set by the various Manage Automations modals.
export interface IFormPolicy extends IPolicy {
  installSoftwareEnabled: boolean;
  swIdToInstall?: number;
  swNameToInstall?: string;
  runScriptEnabled: boolean;
  scriptIdToRun?: number;
  scriptNameToRun?: string;
}

interface IPoliciesPaginatedListProps {
  isSelected: string | ((item: IFormPolicy) => boolean);
  renderItemRow?: (
    item: IFormPolicy,
    onChange: (item: IFormPolicy) => void
  ) => ReactElement | false | null | undefined;
  onToggleItem: (item: IFormPolicy) => IFormPolicy;
  /** A function defining the conditions under which to disable a policy. */
  getPolicyDisabled?: (policy: IFormPolicy) => boolean;
  getPolicyTooltipContent?: (policy: IFormPolicy) => React.ReactNode;
  onCancel: () => void;
  onSubmit: (formData: IFormPolicy[]) => void;
  isUpdating: boolean;
  disableList?: boolean;
  disableSave?: (changedItems: IFormPolicy[]) => boolean | string;
  teamId: number;
  helpText: ReactElement | undefined | null;
}

const baseClass = "policies-paginated-list";

function PoliciesPaginatedList(
  {
    isSelected,
    renderItemRow,
    onToggleItem,
    getPolicyDisabled,
    getPolicyTooltipContent,
    onCancel,
    onSubmit,
    isUpdating,
    disableList = false,
    disableSave,
    teamId,
    helpText,
  }: IPoliciesPaginatedListProps,
  ref: Ref<IPaginatedListHandle<IFormPolicy>>
) {
  const { config } = useContext(AppContext);

  // Create a ref to use with the PaginatedList, so we can access its dirty items.
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);

  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const [saveDisabled, setSaveDisabled] = useState<string | boolean>(false);

  const [pageNumber, setPageNumber] = useState(0);

  // Allow parents access to the `getDirtyItems` of the underlying PaginatedList.
  useImperativeHandle(ref, () => ({
    getDirtyItems() {
      if (paginatedListRef.current) {
        return paginatedListRef.current.getDirtyItems();
      }
      return [];
    },
    reload: () => Promise.resolve(), // not used, but required for the interface
  }));

  // When "save" is clicked, call the parent's `onSubmit` with the set of changed items.
  const onClickSave = () => {
    let changedItems: IFormPolicy[] = [];
    if (paginatedListRef.current) {
      changedItems = paginatedListRef.current.getDirtyItems();
    }

    onSubmit(changedItems);
  };

  // When any item in the list is changed, check whether we should disable the save button.
  const onUpdate = useCallback(
    (changedItems: IFormPolicy[]) => {
      if (!disableSave) {
        return;
      }
      setSaveDisabled(disableSave(changedItems));
    },
    [disableSave]
  );

  // Fetch a single page of policies.
  const DEFAULT_PAGE_SIZE = 10;
  const DEFAULT_SORT_COLUMN = "name";

  // API CALLS
  let policiesQueryKey: IPoliciesApiParams;
  let policiesQueryAPI: (
    params: IPoliciesApiParams
  ) => Promise<ILoadAllPoliciesResponse | ILoadTeamPoliciesResponse>;
  let countQueryKey: IPoliciesCountApiParams;
  let countQueryAPI: (
    params: IPoliciesCountApiParams
  ) => Promise<IPoliciesCountResponse>;
  if (teamId === APP_CONTEXT_ALL_TEAMS_ID) {
    policiesQueryKey = {
      page: pageNumber,
      perPage: DEFAULT_PAGE_SIZE,
      query: "",
      orderDirection: "asc" as const,
      orderKey: DEFAULT_SORT_COLUMN,
      teamId,
    };
    policiesQueryAPI = globalPoliciesAPI.loadAllNew;
    countQueryKey = {
      query: "",
      teamId,
    };
    countQueryAPI = globalPoliciesAPI.getCount;
  } else {
    policiesQueryKey = {
      page: pageNumber,
      perPage: DEFAULT_PAGE_SIZE,
      query: "",
      orderDirection: "asc" as const,
      orderKey: DEFAULT_SORT_COLUMN,
      teamId,
      mergeInherited: false,
    };
    policiesQueryAPI = teamPoliciesAPI.loadAllNew;
    countQueryKey = {
      query: "",
      teamId,
      mergeInherited: false,
    };
    countQueryAPI = teamPoliciesAPI.getCount;
  }

  const { data, isFetching: isLoading } = useQuery<
    ILoadAllPoliciesResponse | ILoadTeamPoliciesResponse,
    Error,
    IFormPolicy[]
  >([policiesQueryKey], () => policiesQueryAPI(policiesQueryKey), {
    keepPreviousData: true,
    select: (
      policiesResponse: ILoadAllPoliciesResponse | ILoadTeamPoliciesResponse
    ) => {
      // Marshall the response into IFormPolicy objects.
      return (policiesResponse.policies || []).map((policy) => {
        return {
          ...policy,
          installSoftwareEnabled: !!policy.install_software,
          swIdToInstall: policy.install_software?.software_title_id,
          runScriptEnabled: !!policy.run_script,
          scriptIdToRun: policy.run_script?.id,
          scriptNameToRun: policy.run_script?.name,
        };
      }) as IFormPolicy[];
    },
  });

  const { data: count, isFetching: isFetchingCount } = useQuery<
    IPoliciesCountResponse,
    Error,
    number
  >([countQueryKey], () => countQueryAPI(countQueryKey), {
    select: (countResponse: IPoliciesCountResponse) => countResponse.count,
  });

  return (
    <div className={`${baseClass} form`}>
      <div className="form-field">
        <PaginatedList<IFormPolicy>
          ref={paginatedListRef}
          data={data || []}
          count={count || 0}
          isLoading={isLoading || isFetchingCount}
          currentPage={pageNumber}
          onChangePage={setPageNumber}
          isSelected={isSelected}
          isItemDisabled={getPolicyDisabled}
          getItemTooltipContent={getPolicyTooltipContent}
          onClickRow={onToggleItem}
          renderItemRow={renderItemRow}
          pageSize={DEFAULT_PAGE_SIZE}
          onUpdate={onUpdate}
          disabled={disableList || gitOpsModeEnabled}
          heading={<span className={`${baseClass}__header`}>Policies</span>}
          helpText={helpText}
        />
      </div>
      <div className="modal-cta-wrap">
        <GitOpsModeTooltipWrapper
          position="top"
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              showArrow
              position="top"
              tipContent={saveDisabled}
              disableTooltip={disableChildren || !saveDisabled}
              underline={false}
            >
              <Button
                type="submit"
                onClick={onClickSave}
                className="save-loading"
                isLoading={isUpdating}
                disabled={!!saveDisabled || disableChildren}
              >
                Save
              </Button>
            </TooltipWrapper>
          )}
        />
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );
}

// Wrap with forwardRef to expose the imperative handle.
// TODO -- can remove this after upgrading to React 19.
export default forwardRef(PoliciesPaginatedList);
