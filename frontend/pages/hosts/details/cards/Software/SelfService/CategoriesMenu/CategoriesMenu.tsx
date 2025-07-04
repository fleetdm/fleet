import React from "react";
import classNames from "classnames";

import LinkWithContext from "components/LinkWithContext";
import TooltipTruncatedText from "components/TooltipTruncatedText";

import { parseHostSoftwareQueryParams } from "../../HostSoftware";
import { ICategory } from "../helpers";

const baseClass = "categories-menu";

export interface ICategoriesMenu {
  categories: ICategory[];
  queryParams: ReturnType<typeof parseHostSoftwareQueryParams>;
  className?: string;
}

const CategoriesMenu = ({
  categories,
  queryParams,
  className,
}: ICategoriesMenu) => {
  const wrapperClasses = classNames(baseClass, className);

  return (
    <ul className={wrapperClasses}>
      {categories.map((cat: ICategory) => {
        const isActive =
          cat.id === queryParams.category_id ||
          (cat.id === 0 && !queryParams.category_id);

        return (
          <li
            className={classNames(`${baseClass}__category-link`, {
              [`${baseClass}__category-link--active`]: isActive,
            })}
          >
            <LinkWithContext
              key={cat.value ?? "all"}
              withParams={{
                type: "query",
                names: ["query", "page", "category_id"],
              }}
              currentQueryParams={{
                ...queryParams,
                page: 0,
                category_id: cat.id !== 0 ? cat.id : undefined,
              }}
              to={location.pathname}
            >
              <TooltipTruncatedText value={cat.label} tooltipPosition="right" />
            </LinkWithContext>
          </li>
        );
      })}
    </ul>
  );
};
export default CategoriesMenu;
