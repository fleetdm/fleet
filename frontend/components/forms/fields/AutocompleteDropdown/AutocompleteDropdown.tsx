import React from "react";
import { Async, OnChangeHandler, Option } from "react-select";
import classnames from "classnames";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import local from "utilities/local";
// @ts-ignore
import debounce from "utilities/debounce";

const baseClass = "autocomplete-dropdown";

interface IAutocompleteDropdown {
  id: string;
  placeholder: string;
  onChange: OnChangeHandler;
  resourceUrl: string;
  valueKey: string;
  labelKey: string;
  value: Option[];
  disabled?: boolean;
  optionComponent?: JSX.Element;
  className?: string;
}

const debounceOptions = {
  timeout: 300,
  leading: false,
  trailing: true,
};

const createUrl = (baseUrl: string, input: string) => {
  return `/api${baseUrl}?query=${input}`;
};

const AutocompleteDropdown = (props: IAutocompleteDropdown): JSX.Element => {
  const {
    className,
    disabled,
    placeholder,
    onChange,
    id,
    resourceUrl,
    valueKey,
    labelKey,
    value,
  } = props;

  const wrapperClass = classnames(baseClass, className);

  const getOptions = debounce((input: string) => {
    if (!input) {
      return Promise.resolve({ options: [] });
    }

    return fetch(createUrl(resourceUrl, input), {
      headers: {
        authorization: `Bearer ${local.getItem("auth_token")}`,
      },
    })
      .then((res) => {
        return res.json();
      })
      .then((json) => {
        return { options: json.users };
      })
      .catch((err) => {
        console.log("There was an error", err);
      });
  }, debounceOptions);

  return (
    <div className={wrapperClass}>
      <Async
        noResultsText={"Nothing found"}
        autoload={false}
        cache={false}
        id={id}
        loadOptions={getOptions}
        disabled={disabled}
        placeholder={placeholder}
        onChange={onChange}
        valueKey={valueKey}
        value={value}
        labelKey={labelKey}
        filterOptions={(options) => options}
        multi
        searchable
      />
    </div>
  );
};

export default AutocompleteDropdown;
