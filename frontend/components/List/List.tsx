import React, { ReactElement } from "react";
import classnames from "classnames";

const baseClass = "list";

type WithIdKey<TKey extends string = "id"> = Record<TKey, React.Key>;

export interface IListProps<TItem, TKey extends string = "id"> {
  data: TItem[];
  isLoading?: boolean;
  idKey?: TKey;
  renderItemRow?: (item: TItem) => ReactElement | false | null | undefined;
  onClickRow?: (item: TItem) => void;
  isRowClickable?: (item: TItem) => boolean;
  heading?: JSX.Element;
  helpText?: React.ReactNode;
}

function List<TItem extends WithIdKey<TKey>, TKey extends string = "id">({
  data,
  isLoading = false,
  idKey: _idKey,
  renderItemRow,
  onClickRow,
  isRowClickable,
  heading,
  helpText,
}: IListProps<TItem, TKey>): JSX.Element {
  const idKey = (_idKey ?? "id") as TKey;

  return (
    <div className={baseClass}>
      {isLoading && <div className="loading-overlay" />}
      <ul className={`${baseClass}__list`}>
        {heading && (
          <li className={`${baseClass}__row ${baseClass}__header`}>
            {heading}
          </li>
        )}
        {data.map((item) => {
          if (!item) return null;

          const clickable = isRowClickable?.(item) ?? !!onClickRow;

          const rowClasses = classnames(`${baseClass}__row`, {
            [`${baseClass}__row--clickable`]: clickable,
          });

          return (
            // eslint-disable-next-line jsx-a11y/no-noninteractive-element-interactions
            <li
              className={rowClasses}
              key={item[idKey]}
              onClick={() => {
                if (clickable) {
                  onClickRow?.(item);
                }
              }}
            >
              {renderItemRow?.(item) ?? null}
            </li>
          );
        })}
      </ul>
      {helpText && <div className="form-field__help-text">{helpText}</div>}
    </div>
  );
}

export default List;
