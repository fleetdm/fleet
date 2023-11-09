import React from "react";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "upload-list";

interface IUploadListProps {
  listItems: any[]; // TODO: typings
  HeadingComponent?: (props: any) => JSX.Element; // TODO: Typings
  ListItemComponent: (props: { listItem: any }) => JSX.Element; // TODO: types
}

const UploadList = ({
  listItems,
  HeadingComponent,
  ListItemComponent,
}: IUploadListProps) => {
  const items = listItems.map((listItem) => {
    return (
      <li key={`${listItem.id}`} className={`${baseClass}__list-item`}>
        <ListItemComponent listItem={listItem} />
      </li>
    );
  });
  return (
    <div className={baseClass}>
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
