/**
 * Used on Software title > CategoriesEndUserExperienceModal and Software title > Edit Appearance Modal
 *
 * Non-mobile preview modal:
 * - uses HTML/CSS instead for maintainability as the self-service UI changes
 * - dynamic name/icon
 *
 * Mobile preview modal:
 * - uses a screenshot
 * - dynamic name/icon/version
 */

import React from "react";
import { noop } from "lodash";
import Card from "components/Card";
import SearchField from "components/forms/fields/SearchField";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import CategoriesMenu from "pages/hosts/details/cards/Software/SelfService/components/CategoriesMenu";
import SelfServiceHeader from "pages/hosts/details/cards/Software/SelfService/components/SelfServiceHeader";
import { CATEGORIES_NAV_ITEMS } from "pages/hosts/details/cards/Software/SelfService/helpers";
import PreviewSelfServiceMobileIcon from "../../../../../../assets/images/preview-self-service-mobile-icon.png";

const baseClass = "self-service-preview";

interface ISelfServicePreviewProps {
  /** iOS/iPadOS uses screenshot + dynamic overlay; otherwise HTML preview */
  isIosOrIpadosApp: boolean;
  /** Shared data for mobile preview */
  contactUrl: string;
  name: string;
  displayName: string;
  versionLabel: string;
  /** What to render for the app icon in the list (img or <SoftwareIcon/>) */
  renderIcon: () => React.ReactNode;
  /** What to render as the “table” area for desktop (e.g. BasicSoftwareTable) */
  renderTable?: () => React.ReactNode;
}

const SelfServicePreview = ({
  isIosOrIpadosApp,
  contactUrl,
  name,
  displayName,
  versionLabel,
  renderIcon,
  renderTable,
}: ISelfServicePreviewProps) => {
  if (isIosOrIpadosApp) {
    // Mobile preview with screenshot + overlay
    return (
      <Card
        borderRadiusSize="medium"
        color="white"
        className={`${baseClass}__preview-card ${baseClass}__preview-card--mobile`}
        paddingSize="xlarge"
      >
        <div className={`${baseClass}__preview-img-container--mobile`}>
          <img
            className={`${baseClass}__preview-img--mobile`}
            src={PreviewSelfServiceMobileIcon}
            alt="Preview icon on Fleet Desktop > Self-service"
          />
        </div>
        <div className={`${baseClass}__self-service-preview--mobile`}>
          {renderIcon()}
          <div
            className={`${baseClass}__self-service-preview-name-version--mobile`}
          >
            <div className={`${baseClass}__self-service-preview-name--mobile`}>
              <TooltipTruncatedText value={displayName || name} />
            </div>
            <div
              className={`${baseClass}__self-service-preview-version--mobile`}
            >
              {versionLabel}
            </div>
          </div>
        </div>
      </Card>
    );
  }

  // Desktop HTML/CSS self-service preview
  return (
    <Card
      borderRadiusSize="medium"
      color="grey"
      className={`${baseClass}__preview-card`}
      paddingSize="xlarge"
    >
      <div className={`${baseClass}__disabled-overlay`} />
      <Card
        className={`${baseClass}__preview-card__self-service`}
        borderRadiusSize="xxlarge"
      >
        <SelfServiceHeader contactUrl={contactUrl} variant="preview" />
        <SearchField placeholder="Search by name" onChange={noop} disabled />
        <div className={`${baseClass}__table`}>
          <CategoriesMenu
            categories={CATEGORIES_NAV_ITEMS}
            queryParams={{
              query: "",
              order_direction: "asc",
              order_key: "name",
              page: 0,
              per_page: 100,
            }}
            readOnly
          />
          {renderTable && renderTable()}
        </div>
      </Card>
    </Card>
  );
};

export default SelfServicePreview;
