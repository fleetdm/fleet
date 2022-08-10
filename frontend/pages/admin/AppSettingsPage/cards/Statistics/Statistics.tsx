import React, { useState } from "react";
import { syntaxHighlight } from "utilities/helpers";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";

import Modal from "components/Modal";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import {
  IAppConfigFormProps,
  IFormField,
  usageStatsPreview,
} from "../constants";

const baseClass = "app-config-form";

const Statistics = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [
    showUsageStatsPreviewModal,
    setShowUsageStatsPreviewModal,
  ] = useState<boolean>(false);
  const [formData, setFormData] = useState<any>({
    enableUsageStatistics: appConfig.server_settings.enable_analytics,
  });

  const { enableUsageStatistics } = formData;

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const toggleUsageStatsPreviewModal = () => {
    setShowUsageStatsPreviewModal(!showUsageStatsPreviewModal);
    return false;
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      server_settings: {
        server_url: appConfig.server_settings.server_url || "",
        live_query_disabled:
          appConfig.server_settings.live_query_disabled || false,
        enable_analytics: enableUsageStatistics,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const renderUsageStatsPreviewModal = () => {
    if (!showUsageStatsPreviewModal) {
      return null;
    }

    return (
      <Modal
        title="Usage statistics"
        onExit={toggleUsageStatsPreviewModal}
        className={`${baseClass}__usage-stats-preview-modal`}
      >
        <>
          <p>An example JSON payload sent to Fleet Device Management Inc.</p>
          <pre
            dangerouslySetInnerHTML={{
              __html: syntaxHighlight(usageStatsPreview),
            }}
          />
          <div className="flex-end">
            <Button type="button" onClick={toggleUsageStatsPreviewModal}>
              Done
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  return (
    <>
      <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
        <div className={`${baseClass}__section`}>
          <h2>Usage statistics</h2>
          <p className={`${baseClass}__section-description`}>
            Help improve Fleet by sending usage statistics.
            <br />
            <br />
            This information helps our team better understand feature adoption
            and usage, and allows us to see how Fleet is adding value, so that
            we can make better product decisions.
            <br />
            <br />
            <a
              href="https://fleetdm.com/docs/using-fleet/usage-statistics#usage-statistics"
              className={`${baseClass}__learn-more`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about usage statistics&nbsp;
              <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </p>
          <div className={`${baseClass}__inputs ${baseClass}__inputs--usage`}>
            <Checkbox
              onChange={handleInputChange}
              name="enableUsageStatistics"
              value={enableUsageStatistics}
              parseTarget
            >
              Enable usage statistics
            </Checkbox>
          </div>
          <div className={`${baseClass}__inputs ${baseClass}__inputs--preview`}>
            <Button
              type="button"
              variant="inverse"
              onClick={toggleUsageStatsPreviewModal}
            >
              Preview payload
            </Button>
          </div>
        </div>
        <Button
          type="submit"
          variant="brand"
          className="save-loading"
          isLoading={isUpdatingSettings}
        >
          Save
        </Button>
      </form>
      {renderUsageStatsPreviewModal()}
    </>
  );
};

export default Statistics;
