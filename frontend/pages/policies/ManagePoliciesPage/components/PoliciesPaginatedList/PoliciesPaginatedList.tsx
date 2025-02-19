import React, {
  useCallback,
  useImperativeHandle,
  useRef,
  forwardRef,
  Ref,
} from "react";
import { ReactElement } from "react-markdown/lib/react-markdown";
import PaginatedList, { IPaginatedListHandle } from "components/PaginatedList";
import { useQueryClient } from "react-query";
import { IPolicy } from "interfaces/policy";
import teamPoliciesAPI, {
  ITeamPoliciesQueryKey,
} from "services/entities/team_policies";
import globalPoliciesAPI, {
  IPoliciesQueryKey,
} from "services/entities/global_policies";

import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Button from "components/buttons/Button";

// Extend the IPolicy interface with some virtual properties that make it easier
// to track item state. These are set by the various Manage Automations modals.
export interface IFormPolicy extends IPolicy {
  installSoftwareEnabled: boolean;
  swIdToInstall?: number;
  runScriptEnabled: boolean;
  scriptIdToRun?: number;
}

interface IPoliciesPaginatedListProps {
  isSelected: string | ((item: IFormPolicy) => boolean);
  renderItemRow?: (
    item: IFormPolicy,
    onChange: (item: IFormPolicy) => void
  ) => ReactElement | false | null | undefined;
  onToggleItem: (item: IFormPolicy) => IFormPolicy;
  onCancel: () => void;
  onSubmit: (formData: IFormPolicy[]) => void;
  isUpdating: boolean;
  teamId: number;
  footer: ReactElement | undefined | null;
}

const baseClass = "policies-paginated-list";

function PoliciesPaginatedList(
  {
    isSelected,
    renderItemRow,
    onToggleItem,
    onCancel,
    onSubmit,
    isUpdating,
    teamId,
    footer,
  }: IPoliciesPaginatedListProps,
  ref: Ref<IPaginatedListHandle<IFormPolicy>>
) {
  // Create a ref to use with the PaginatedList, so we can access its dirty items.
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);

  // Allow parents access to the `getDirtyItems` of the underlying PaginatedList.
  useImperativeHandle(ref, () => ({
    getDirtyItems() {
      if (paginatedListRef.current) {
        return paginatedListRef.current.getDirtyItems();
      }
      return [];
    },
  }));

  // When "save" is clicked, call the parent's `onSubmit` with the set of changed items.
  const onClickSave = () => {
    let changedItems: IFormPolicy[] = [];
    if (paginatedListRef.current) {
      changedItems = paginatedListRef.current.getDirtyItems();
    }
    onSubmit(changedItems);
  };

  // Fetch a single page of policies.
  const queryClient = useQueryClient();
  const DEFAULT_PAGE_SIZE = 8;
  const DEFAULT_SORT_COLUMN = "name";

  const fetchPage = useCallback((pageNumber: number) => {
    let fetchPromise;

    if (teamId === APP_CONTEXT_ALL_TEAMS_ID) {
      fetchPromise = queryClient.fetchQuery(
        [
          {
            scope: "globalPolicies",
            page: pageNumber,
            perPage: DEFAULT_PAGE_SIZE,
            query: "",
            orderDirection: "asc" as const,
            orderKey: DEFAULT_SORT_COLUMN,
          },
        ],
        ({ queryKey }) => {
          return globalPoliciesAPI.loadAllNew(queryKey[0]);
        }
      );
    } else {
      fetchPromise = queryClient.fetchQuery(
        [
          {
            scope: "teamPolicies",
            page: pageNumber,
            perPage: DEFAULT_PAGE_SIZE,
            query: "",
            orderDirection: "asc" as const,
            orderKey: DEFAULT_SORT_COLUMN,
            teamId,
            mergeInherited: false,
          },
        ],
        ({ queryKey }) => {
          return teamPoliciesAPI.loadAllNew(queryKey[0]);
        }
      );
    }

    return fetchPromise.then((policiesResponse) => {
      // Marshall the response into IFormPolicy objects.
      return (policiesResponse.policies || []).map((policy) => ({
        ...policy,
        installSoftwareEnabled: !!policy.install_software,
        swIdToInstall: policy.install_software?.software_title_id,
        runScriptEnabled: !!policy.run_script,
        scriptIdToRun: policy.run_script?.id,
      })) as IFormPolicy[];
    });
  }, []);

  return (
    <div className={`${baseClass} form`}>
      <div className="form-field">
        <div className="form-field__label">Policies:</div>
        <div>
          <PaginatedList<IFormPolicy>
            ref={paginatedListRef}
            fetchPage={fetchPage}
            isSelected={isSelected}
            onToggleItem={onToggleItem}
            renderItemRow={renderItemRow}
            totalItems={100}
            pageSize={DEFAULT_PAGE_SIZE}
          />

          {footer}
        </div>
      </div>
      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onClickSave}
          className="save-loading"
          isLoading={isUpdating}
        >
          Save
        </Button>
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );
}

export default forwardRef(PoliciesPaginatedList);
