import React, {
  useState,
  useEffect,
  useImperativeHandle,
  forwardRef,
  Ref,
} from "react";
import { ReactElement } from "react-markdown/lib/react-markdown";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Pagination from "components/Pagination";

const baseClass = "paginated-list";

export interface IPaginatedListHandle<TItem> {
  getDirtyItems: () => TItem[];
}
interface IPaginatedListProps<TItem> {
  fetchPage: (pageNumber: number) => Promise<TItem[]>;
  idKey?: string; // defaults to `id`
  labelKey?: string;
  isSelected: string | ((item: TItem) => boolean);
  renderItemLabel?: (item: TItem) => ReactElement | false | null | undefined;
  renderItemRow?: (item: TItem, onChange(item: TItem)) => ReactElement | false | null | undefined;
  onToggleItem: (item: TItem) => TItem;
  pageSize?: number;
  totalItems: number;
}

function PaginatedListInner<TItem extends Record<string, any>>(
  {
    fetchPage,
    idKey: _idKey,
    labelKey: _labelKey,
    pageSize: _pageSize,
    renderItemLabel,
    renderItemRow,
    onToggleItem,
    isSelected,
  }: IPaginatedListProps<TItem>,
  ref: Ref<IPaginatedListHandle<TItem>>
) {
  const [currentPage, setCurrentPage] = useState(0);
  const [items, setItems] = useState<TItem[]>([]);
  const [dirtyItems, setDirtyItems] = useState<Record<string | number, TItem>>(
    {}
  );
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const idKey = _idKey ?? "id";
  const labelKey = _labelKey ?? "name";
  const pageSize = _pageSize ?? 20;

  console.log(dirtyItems);

  useEffect(() => {
    let isCancelled = false;

    async function loadPage() {
      try {
        setIsLoading(true);
        setError(null);
        const result = await fetchPage(currentPage);
        if (!isCancelled) {
          setItems(result);
        }
      } catch (err) {
        if (!isCancelled) {
          setError(err as Error);
        }
      } finally {
        if (!isCancelled) {
          setIsLoading(false);
        }
      }
    }

    loadPage();

    return () => {
      isCancelled = true;
    };
  }, [currentPage, fetchPage]);

  // If you have an imperative API (e.g. getDirtyItems), define it:
  useImperativeHandle(ref, () => ({
    getDirtyItems() {
      return []; // TODO: return any "dirty" items
    },
  }));

  if (isLoading) return <p>Loading...</p>;
  if (error) return <p>Error: {error.message}</p>;

  return (
    <div>
      <ul className={`${baseClass}__list`}>
        {items.map((_item) => {
          const item = dirtyItems[_item[idKey]] ?? _item;
          return (
            <li className={`${baseClass}__row`} key={item[idKey]}>
              <Checkbox
                value={
                  typeof isSelected === "function"
                    ? isSelected(item)
                    : !!item[isSelected]
                }
                name={`item_${item[idKey]}_checkbox`}
                onChange={() => {
                  setDirtyItems({
                    ...dirtyItems,
                    [item[idKey]]: onToggleItem(item),
                  });
                }}
              >
                {renderItemLabel ? (
                  renderItemLabel(item)
                ) : (
                  <span>{item[labelKey]}</span>
                )}
              </Checkbox>
              {renderItemRow && renderItemRow(item)}
            </li>
          );
        })}
      </ul>
      <Pagination
        resultsOnCurrentPage={items.length}
        currentPage={currentPage}
        resultsPerPage={pageSize}
        onPaginationChange={setCurrentPage}
      />
    </div>
  );
}

// Wrap with forwardRef to expose the imperative handle:
const PaginatedList = forwardRef(PaginatedListInner) as <TItem>(
  props: IPaginatedListProps<TItem> & {
    ref?: Ref<IPaginatedListHandle<TItem>>;
  }
) => JSX.Element;

export default PaginatedList;
