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
import teamPoliciesAPI from "services/entities/team_policies";
import Button from "components/buttons/Button";

export interface IFormPolicy extends IPolicy {
  installSoftwareEnabled: boolean;
  swIdToInstall?: number;
  runScriptEnabled: boolean;
  scriptIdToRun?: number;
}

interface IPoliciesPaginatedListProps {
  isSelected: string | ((item: IFormPolicy) => boolean);
  renderItemLabel?: (item: IFormPolicy) => ReactElement | null;
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
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);

  // Expose our own handle to the parent
  useImperativeHandle(ref, () => ({
    getDirtyItems() {
      if (paginatedListRef.current) {
        return paginatedListRef.current.getDirtyItems();
      }
      return [];
    },
  }));

  const onClickSave = () => {
    let changedItems: IFormPolicy[] = [];
    if (paginatedListRef.current) {
      changedItems = paginatedListRef.current.getDirtyItems();
    }
    onSubmit(changedItems);
  };

  const queryClient = useQueryClient();
  const DEFAULT_PAGE_SIZE = 10;
  const DEFAULT_SORT_COLUMN = "name";

  const fetchPage = useCallback((pageNumber: number) => {
    return queryClient
      .fetchQuery(
        [
          {
            scope: "teamPolicies",
            page: pageNumber,
            perPage: DEFAULT_PAGE_SIZE,
            query: "",
            orderDirection: "asc" as const,
            orderKey: DEFAULT_SORT_COLUMN,
            // teamIdForApi will never actually be undefined here
            teamId,
            // no teams does inherit
            mergeInherited: false,
          },
        ],
        ({ queryKey }) => {
          return teamPoliciesAPI.loadAllNew(queryKey[0]);
        }
      )
      .then((policiesResponse) => {
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
            pageSize={10}
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
