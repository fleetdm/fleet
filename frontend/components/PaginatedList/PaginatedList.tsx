import React, {
  useState,
  useEffect,
  useImperativeHandle,
  forwardRef,
  Ref,
} from "react";
import classnames from "classnames";
import { ReactElement } from "react-markdown/lib/react-markdown";
import Checkbox from "components/forms/fields/Checkbox";
import Spinner from "components/Spinner";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import Pagination from "components/Pagination";

const baseClass = "paginated-list";

// Create an interface for the Ref, so that when parents use `useRef` to provide
// a reference to this list, they can call `getDirtyItems()` on it to retrieve
// the list of dirty items.
export interface IPaginatedListHandle<TItem> {
  getDirtyItems: () => TItem[];
}
interface IPaginatedListProps<TItem> {
  // Function to fetch one page of data.
  // Parents should memoize this function with useCallback() so that
  // it is only called when needed.
  fetchPage: (pageNumber: number) => Promise<TItem[]>;
  // Function to fetch the total # of items.
  // Parents should memoize this function with useCallback() so that
  // it is only called when needed.
  fetchCount?: () => Promise<number>;
  // UID property in an item. Defaults to `id`.
  idKey?: string;
  // Property to use as an item's label. Defaults to `name`.
  labelKey?: string;
  // How to determine whether to check an item's checkbox.
  // If string, a key in an item whose truthiness will be checked.
  // if function, a function that given an item, returns a boolean.
  isSelected: string | ((item: TItem) => boolean);
  // Custom function to render the label for an item.
  renderItemLabel?: (item: TItem) => ReactElement | null;
  // Custom function to render extra markup (besides the label) in an item row.
  renderItemRow?: (
    item: TItem,
    // A callback function that the extra markup logic can call to indicate a change
    // to the item, for example if a dropdown is changed.
    onChange: (item: TItem) => void
  ) => ReactElement | false | null | undefined;
  // A function to call when an item's checkbox is toggled.
  // Parents can use this to change whatever item metadata is needed to toggle
  // the value indicated by `isSelected`.
  onToggleItem: (item: TItem) => TItem;
  // The size of the page to fetch and show.
  pageSize?: number;
  // An optional header component.
  heading?: JSX.Element;
  // A function to call when the list of dirty items changes.
  onUpdate?: (changedItems: TItem[]) => void;
  // Whether the list should be disabled.
  disabled?: boolean;
}

function PaginatedListInner<TItem extends Record<string, any>>(
  {
    fetchPage,
    fetchCount,
    idKey: _idKey,
    labelKey: _labelKey,
    pageSize: _pageSize,
    renderItemLabel,
    renderItemRow,
    onToggleItem,
    onUpdate,
    isSelected,
    disabled = false,
    heading,
  }: IPaginatedListProps<TItem>,
  ref: Ref<IPaginatedListHandle<TItem>>
) {
  // The # of the page to display.
  const [currentPage, setCurrentPage] = useState(0);
  // The set of items fetched via `fetchPage`.
  const [items, setItems] = useState<TItem[]>([]);
  // The total # of items fetched via `fetchCount`.
  const [totalItems, setTotalItems] = useState(0);
  // The set of items that have been changed in some way.
  const [dirtyItems, setDirtyItems] = useState<Record<string | number, TItem>>(
    {}
  );
  const [isLoadingPage, setIsLoadingPage] = useState(false);
  const [isLoadingCount, setIsLoadingCount] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const idKey = _idKey ?? "id";
  const labelKey = _labelKey ?? "name";
  const pageSize = _pageSize ?? 20;

  // When the current page # changes, fetch a new page of data.
  useEffect(() => {
    let isCancelled = false;

    async function loadPage() {
      try {
        setIsLoadingPage(true);
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
          setIsLoadingPage(false);
        }
      }
    }

    loadPage();

    return () => {
      isCancelled = true;
    };
  }, [currentPage, fetchPage]);

  // Fetch the total # of items.
  // This will generally only happen once, assuming the parent
  // uses useCallback() to memoize the fetchCount function.
  // To retrigger this (for example, after an item is added or removed),
  // the parent can add dependencies to the useCallback().
  useEffect(() => {
    let isCancelled = false;

    async function loadCount() {
      try {
        if (!fetchCount) {
          return;
        }
        setIsLoadingCount(true);
        const result = await fetchCount();
        if (!isCancelled) {
          setTotalItems(result);
        }
      } catch (err) {
        if (!isCancelled) {
          setError(err as Error);
        }
      } finally {
        if (!isCancelled) {
          setIsLoadingCount(false);
        }
      }
    }

    loadCount();

    return () => {
      isCancelled = true;
    };
  }, [fetchCount]);

  // Whenever the dirty items list changes, notify the parent.
  useEffect(() => {
    if (onUpdate) {
      onUpdate(Object.values(dirtyItems));
    }
  }, [onUpdate, dirtyItems]);

  // Create an imperative handle for this component so that parents
  // can call `ref.current.getDirtyItems()` to get the changed set.
  useImperativeHandle(ref, () => ({
    getDirtyItems() {
      return Object.values(dirtyItems);
    },
  }));

  const disableNext = () => {
    if (!totalItems) {
      return items.length < pageSize;
    }
    return currentPage * pageSize + items.length >= totalItems;
  };

  // TODO -- better error state?
  if (error) return <p>Error: {error.message}</p>;

  // Render the list.
  const classes = classnames(baseClass, "form", {
    "form-fields--disabled": disabled,
  });
  return (
    <div className={classes}>
      {(isLoadingPage || isLoadingCount) && (
        <div className="loading-overlay">
          <Spinner />
        </div>
      )}
      <div>
        <ul className={`${baseClass}__list`}>
          {heading && (
            <li className={`${baseClass}__row ${baseClass}__header`}>
              {heading}
            </li>
          )}
          {items.map((_item) => {
            // If an item has been marked as changed, use the changed version
            // of the item rather than the one from the page fetch.  This allows
            // us to render an item correctly even after we've navigated away
            // from its page and then back again.
            const item = dirtyItems[_item[idKey]] ?? _item;
            return (
              // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions
              <li
                className={`${baseClass}__row`}
                key={item[idKey]}
                onClick={() => {
                  // When checkbox is toggled, set item as dirty.
                  // The parent is responsible for actually updating item properties via onToggleItem().
                  setDirtyItems({
                    ...dirtyItems,
                    [item[idKey]]: onToggleItem(item),
                  });
                }}
              >
                <Checkbox
                  disabled={disabled}
                  value={
                    typeof isSelected === "function"
                      ? isSelected(item)
                      : item[isSelected]
                  }
                  name={`item_${item[idKey]}_checkbox`}
                >
                  {renderItemLabel ? (
                    renderItemLabel(item)
                  ) : (
                    <TooltipTruncatedText value={<>{item[labelKey]}</>} />
                  )}
                </Checkbox>
                {renderItemRow &&
                  // If a custom row renderer was supplied, call it with the item value
                  // as well as the callback the parent can use to indicate changes to an item.
                  renderItemRow(item, (changedItem) => {
                    setDirtyItems({
                      ...dirtyItems,
                      [changedItem[idKey]]: changedItem,
                    });
                  })}
              </li>
            );
          })}
        </ul>
        <Pagination
          disablePrev={currentPage === 0}
          disableNext={disableNext()}
          onNextPage={() => setCurrentPage(currentPage + 1)}
          onPrevPage={() => setCurrentPage(currentPage - 1)}
          hidePagination={currentPage === 0 && disableNext()}
        />
      </div>
    </div>
  );
}

// Wrap with forwardRef to expose the imperative handle.
// TODO -- can remove this after upgrading to React 19.
const PaginatedList = forwardRef(PaginatedListInner) as <TItem>(
  props: IPaginatedListProps<TItem> & {
    ref?: Ref<IPaginatedListHandle<TItem>>;
  }
) => JSX.Element;

export default PaginatedList;
