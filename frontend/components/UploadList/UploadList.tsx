import React from "react";
import classnames from "classnames";

const baseClass = "upload-list";

interface IUploadListProps<T = any> {
  /** The attribute name that is used for the react key for each list item */
  keyAttribute: keyof T;
  listItems: T[];
  HeadingComponent?: (props: any) => JSX.Element;
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
        key={`${listItem[keyAttribute]}`}
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
