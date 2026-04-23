import React, { useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IAppConfigFormProps } from "../constants";

const baseClass = "app-config-form";

interface IDataCollectionFormData {
  dataCollectionUptime: boolean;
  dataCollectionCve: boolean;
}

const DataCollection = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IDataCollectionFormData>({
    dataCollectionUptime: appConfig.features.data_collection.uptime,
    dataCollectionCve: appConfig.features.data_collection.cve,
  });

  const { dataCollectionUptime, dataCollectionCve } = formData;

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    setFormData({ ...formData, [name]: value });
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    handleSubmit({
      features: {
        data_collection: {
          uptime: dataCollectionUptime,
          cve: dataCollectionCve,
        },
      },
    });
  };

  return (
    <SettingsSection title="Data collection">
      <PageDescription
        variant="right-panel"
        content={
          <p className={`${baseClass}__section-description`}>
            Turn on/off data collection for charts that appear on the dashboard.
          </p>
        }
      />
      <form onSubmit={onFormSubmit} autoComplete="off">
        <Checkbox
          onChange={onInputChange}
          name="dataCollectionUptime"
          value={dataCollectionUptime}
          parseTarget
        >
          Hosts active
        </Checkbox>
        <Checkbox
          onChange={onInputChange}
          name="dataCollectionCve"
          value={dataCollectionCve}
          parseTarget
        >
          Vulnerabilities
        </Checkbox>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren}
              className="button-wrap"
              isLoading={isUpdatingSettings}
            >
              Save
            </Button>
          )}
        />
      </form>
    </SettingsSection>
  );
};

export default DataCollection;
