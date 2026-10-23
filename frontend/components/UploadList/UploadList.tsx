import classnames from "classnames";
import React from "react";

// TODO: rename this component (and its `upload-list` BEM class) — the name is
// a historical artifact from when it only backed an upload picker. It's now a
// general-purpose "icon + name" list (self-service categories, patch deadline
// apps, …) and the mismatch confuses readers. Keeping the rename as a separate
// pass because it ripples through every consumer's SCSS selectors.
const baseClass = "upload-list";

interface IUploadListProps<T = any> {
  /** The attribute name that is used for the react key for each list item.
   * This is optional and not needed when the listItem type is a string
   */
  keyAttribute?: keyof T;
  listItems: T[];
  HeadingComponent?: (props: any) => JSX.Element;
  /** If the row renders user-typed text (name/title/label/description),
   * wrap it in <TooltipTruncatedText /> with `flex: 1; min-width: 0` on the
   * container — see "Lists & rows" in .claude/rules/fleet-frontend.md. */
  ListItemComponent: (props: { listItem: T }) => JSX.Element;
  className?: string;
}

const UploadList = <T,>({
  keyAttribute,
  listItems,
  HeadingComponent,
  ListItemComponent,
  className,
}: IUploadListProps<T>) => {
  const items = listItems.map((listItem) => {
    return (
      <li
        key={keyAttribute ? `${listItem[keyAttribute]}` : `${listItem}`}
        className={`${baseClass}__list-item`}
      >
        <ListItemComponent listItem={listItem} />
      </li>
    );
  });

  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      {HeadingComponent && (
        <div className={`${baseClass}__header`}>
          <HeadingComponent />
        </div>
      )}
      <ul className={`${baseClass}__list`}>{items}</ul>
    </div>
  );
};

export default UploadList;
